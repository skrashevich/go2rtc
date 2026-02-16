package api

import (
	"path/filepath"
	"strings"

	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/AlexxIT/go2rtc/www"
)

// contentTypeByExtension returns the correct Content-Type for a file extension
func contentTypeByExtension(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "" // Don't override Content-Type for unknown extensions
	}
}

// responseWriter wraps http.ResponseWriter to override Content-Type header
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
	path        string
	redirected  bool
}

func fallbackIndexPath(requestPath string) (string, bool) {
	if requestPath == "" || requestPath == "/" {
		return "/index.html", true
	}
	if strings.HasSuffix(requestPath, "/") {
		return requestPath + "index.html", true
	}
	return "", false
}

func shouldFallbackStatus(code int) bool {
	switch code {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound:
		return true
	default:
		return false
	}
}

func newResponseWriter(w http.ResponseWriter, path string) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		path:           path,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code

	// For directory requests, fallback to {dir}/index.html on specific upstream errors.
	// Examples: "/" => "/index.html", "/foo/" => "/foo/index.html"
	if fallbackPath, isDirRequest := fallbackIndexPath(rw.path); isDirRequest && shouldFallbackStatus(code) && !rw.redirected {
		log.Trace().Str("path", rw.path).Str("fallback", fallbackPath).Int("status", code).Msg("[api] directory path error, redirecting to index")

		h := rw.ResponseWriter.Header()
		h.Del("Content-Length")
		h.Del("Transfer-Encoding")
		h.Del("Content-Encoding")
		h.Del("Content-Security-Policy")
		h.Del("X-Content-Type-Options")
		h.Del("X-Frame-Options")
		h.Del("Cross-Origin-Resource-Policy")
		h.Set("Location", fallbackPath)
		h.Set("Content-Type", "text/html; charset=utf-8")

		rw.redirected = true
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	// Override Content-Type for static files from remote proxy
	// This is needed because GitHub raw.githubusercontent.com returns text/plain
	if contentType := contentTypeByExtension(rw.path); contentType != "" {
		rw.ResponseWriter.Header().Set("Content-Type", contentType)
		log.Trace().Str("path", rw.path).Str("content_type", contentType).Msg("[api] overrode Content-Type")
	}

	// Remove restrictive CSP headers from GitHub that block our web UI
	// GitHub returns: Content-Security-Policy: default-src 'none'; sandbox
	// This blocks CSS loading and script execution
	rw.ResponseWriter.Header().Del("Content-Security-Policy")
	rw.ResponseWriter.Header().Del("X-Content-Type-Options")
	rw.ResponseWriter.Header().Del("X-Frame-Options")
	rw.ResponseWriter.Header().Del("Cross-Origin-Resource-Policy")

	log.Trace().Str("path", rw.path).Msg("[api] removed restrictive CSP headers")

	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Swallow upstream body after redirect to avoid short-write errors.
	if rw.redirected {
		return len(b), nil
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func initStatic(staticDir string) {
	base := len(basePath)
	log.Debug().Str("static_dir", staticDir).Msg("[api] initStatic")

	if staticDir != "" {
		// Check if staticDir is a remote HTTP/HTTPS URL
		if isRemoteURL(staticDir) {
			log.Info().Str("url", staticDir).Msg("[api] serve static from remote")
			fileServer := newReverseProxy(staticDir)
			HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
				originalPath := r.URL.Path
				if base > 0 {
					r.URL.Path = r.URL.Path[base:]
					log.Debug().Str("original", originalPath).Str("adjusted", r.URL.Path).Msg("[api] static path adjusted for basePath")
				}
				log.Trace().Str("method", r.Method).Str("path", r.URL.Path).Str("remote", staticDir).Msg("[api] proxying static request")

				// Wrap response writer to set correct Content-Type
				rw := newResponseWriter(w, r.URL.Path)
				fileServer.ServeHTTP(rw, r)
			})
			return
		}

		// Local directory (existing behavior)
		log.Info().Str("dir", staticDir).Msg("[api] serve static")
		fileServer := http.FileServer(http.Dir(staticDir))
		HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
			originalPath := r.URL.Path
			if base > 0 {
				r.URL.Path = r.URL.Path[base:]
				log.Trace().Str("original", originalPath).Str("adjusted", r.URL.Path).Msg("[api] static path adjusted for basePath")
			}
			log.Trace().Str("method", r.Method).Str("path", r.URL.Path).Msg("[api] serving static file")
			fileServer.ServeHTTP(w, r)
		})
		return
	}

	// Embedded filesystem (existing behavior)
	log.Debug().Msg("[api] using embedded static filesystem")
	fileServer := http.FileServer(http.FS(www.Static))
	HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		originalPath := r.URL.Path
		if base > 0 {
			r.URL.Path = r.URL.Path[base:]
			log.Trace().Str("original", originalPath).Str("adjusted", r.URL.Path).Msg("[api] static path adjusted for basePath")
		}
		log.Trace().Str("method", r.Method).Str("path", r.URL.Path).Msg("[api] serving embedded static file")
		fileServer.ServeHTTP(w, r)
	})
}

// isRemoteURL checks if the path is an HTTP/HTTPS URL
func isRemoteURL(path string) bool {
	isRemote := len(path) >= 7 && path[:7] == "http://" || len(path) >= 8 && path[:8] == "https://"
	log.Debug().Str("path", path).Bool("is_remote", isRemote).Msg("[api] isRemoteURL check")
	return isRemote
}

// newReverseProxy creates a reverse proxy for serving static files from a remote URL
func newReverseProxy(target string) http.Handler {
	log.Debug().Str("target", target).Msg("[api] creating reverse proxy")

	u, err := url.Parse(target)
	if err != nil {
		log.Error().Err(err).Str("url", target).Msg("[api] failed to parse remote static URL")
		// Fallback to embedded filesystem
		return http.FileServer(http.FS(www.Static))
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	// Add error handler for proxy errors
	originalErrorHandler := proxy.ErrorHandler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Str("target", target).Msg("[api] proxy error")
		if originalErrorHandler != nil {
			originalErrorHandler(w, r, err)
		} else {
			http.Error(w, "Failed to proxy request", http.StatusBadGateway)
		}
	}

	// Add director logging
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		// Save original host before director modifies it
		originalHost := r.Host
		log.Trace().Str("original_path", r.URL.Path).Str("original_host", originalHost).Msg("[api] proxy director called")

		originalDirector(r)

		// Set r.Host to match r.URL.Host (target host)
		// This is required for services like GitHub raw.githubusercontent.com
		// which validate the Host header matches the request domain
		r.Host = r.URL.Host
		// Store original host in X-Original-Host for reference
		r.Header.Set("X-Original-Host", originalHost)
		r.Header.Set("X-Forwarded-Host", originalHost)

		log.Trace().Str("target_scheme", r.URL.Scheme).Str("target_host", r.URL.Host).Str("target_path", r.URL.Path).Str("final_host", r.Host).Msg("[api] proxy director target")
	}

	return proxy
}
