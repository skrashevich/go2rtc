package api

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/goccy/go-json"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/rs/zerolog"
)

func Init() {
	var cfg struct {
		Mod struct {
			Listen     string `yaml:"listen"`
			Username   string `yaml:"username"`
			Password   string `yaml:"password"`
			BasePath   string `yaml:"base_path"`
			StaticDir  string `yaml:"static_dir"`
			Origin     string `yaml:"origin"`
			TLSListen  string `yaml:"tls_listen"`
			TLSCert    string `yaml:"tls_cert"`
			TLSKey     string `yaml:"tls_key"`
			UnixListen string `yaml:"unix_listen"`
		} `yaml:"api"`
	}

	// default config
	cfg.Mod.Listen = ":1984"

	// load config from YAML
	app.LoadConfig(&cfg)

	if cfg.Mod.Listen == "" && cfg.Mod.UnixListen == "" && cfg.Mod.TLSListen == "" {
		return
	}

	basePath = cfg.Mod.BasePath
	log = app.GetLogger("api")

	initStatic(cfg.Mod.StaticDir)

	HandleFunc("api", apiHandler)
	HandleFunc("api/config", configHandler)
	HandleFunc("api/exit", exitHandler)
	HandleFunc("api/restart", restartHandler)
	HandleFunc("api/log", logHandler)

	Handler = http.DefaultServeMux // 4th

	if cfg.Mod.Origin == "*" {
		Handler = middlewareCORS(Handler) // 3rd
	}

	if cfg.Mod.Username != "" {
		Handler = middlewareAuth(cfg.Mod.Username, cfg.Mod.Password, Handler) // 2nd
	}

	if log.Trace().Enabled() {
		Handler = middlewareLog(Handler) // 1st
	}

	if cfg.Mod.Listen != "" {
		go listen("tcp", cfg.Mod.Listen)
	}

	if cfg.Mod.UnixListen != "" {
		_ = syscall.Unlink(cfg.Mod.UnixListen)
		go listen("unix", cfg.Mod.UnixListen)
	}

	// Initialize the HTTPS server
	if cfg.Mod.TLSListen != "" && cfg.Mod.TLSCert != "" && cfg.Mod.TLSKey != "" {
		go tlsListen("tcp", cfg.Mod.TLSListen, cfg.Mod.TLSCert, cfg.Mod.TLSKey)
	}
}

func listen(network, address string) {
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Error().Err(err).Msg("[api] listen")
		return
	}

	log.Info().Str("addr", address).Msg("[api] listen")

	if network == "tcp" {
		Port = ln.Addr().(*net.TCPAddr).Port
	}

	server := http.Server{
		Handler:           Handler,
		ReadHeaderTimeout: 5 * time.Second, // Example: Set to 5 seconds
	}
	if err = server.Serve(ln); err != nil {
		log.Fatal().Err(err).Msg("[api] serve")
	}
}

func LoadCertificate(certFile, keyFile string) (tls.Certificate, error) {
	var err error
	var cert tls.Certificate
	if strings.IndexByte(certFile, '\n') < 0 && strings.IndexByte(keyFile, '\n') < 0 {
		// check if file path
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	} else {
		// if text file content
		cert, err = tls.X509KeyPair([]byte(certFile), []byte(keyFile))
	}

	return cert, err
}

func tlsListen(network, address, certFile, keyFile string) {
	log.Trace().Str("address", address).Msg("[api] tls listen")
	cert, err := LoadCertificate(certFile, keyFile)
	if err != nil {
		log.Error().Err(err).Caller().Send()
	}
	ln, err := net.Listen(network, address)
	if err != nil {
		log.Error().Err(err).Msg("[api] tls listen")
		return
	}

	certInfo, err := x509.ParseCertificate(cert.Certificate[0])

	if err != nil {
		log.Error().Err(err).Caller().Send()
		return
	}

	tlsExpire := certInfo.NotAfter
	checkCertExpiration(tlsExpire, address)

	server := &http.Server{
		Handler:           Handler,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}},
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err = server.ServeTLS(ln, "", ""); err != nil {
		log.Fatal().Err(err).Msg("[api] tls serve")
	}
}

// checkCertExpiration logs the certificate expiration status.
func checkCertExpiration(expirationTime time.Time, address string) (int, time.Duration) {
	now := time.Now()
	switch {
	case now.Unix()-expirationTime.Unix() > 0 && now.Unix()-expirationTime.Unix() < int64(time.Hour.Seconds()*24):
		log.Warn().Str("ExpireDate", expirationTime.Local().String()).Str("listen addr", address).Msg("[api] tls cert will expire today")
		return 1, time.Until(expirationTime)
	case expirationTime.Before(now):
		log.Error().Str("ExpireDate", expirationTime.Local().String()).Str("listen addr", address).Msg("[api] tls cert expired")
		return -1, time.Until(expirationTime)
	default:
		log.Info().Str("ExpireDate", expirationTime.Local().String()).Str("listen addr", address).Msg("[api] tls")
		return 0, time.Until(expirationTime)
	}
}

var Port int

const (
	MimeJSON = "application/json"
	MimeText = "text/plain"
)

var Handler http.Handler

// HandleFunc handle pattern with relative path:
// - "api/streams" => "{basepath}/api/streams"
// - "/streams"    => "/streams"
func HandleFunc(pattern string, handler http.HandlerFunc) {
	if len(pattern) == 0 || pattern[0] != '/' {
		pattern = basePath + "/" + pattern
	}
	log.Trace().Str("path", pattern).Msg("[api] register path")
	http.HandleFunc(pattern, handler)
}

// ResponseJSON important always add Content-Type
// so go won't need to call http.DetectContentType
func ResponseJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", MimeJSON)
	_ = json.NewEncoder(w).Encode(v)
}

func ResponsePrettyJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", MimeJSON)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func Response(w http.ResponseWriter, body any, contentType string) {
	w.Header().Set("Content-Type", contentType)

	switch v := body.(type) {
	case []byte:
		_, _ = w.Write(v)
	case string:
		_, _ = w.Write([]byte(v))
	default:
		_, _ = fmt.Fprint(w, body)
	}
}

const StreamNotFound = "stream not found"

var basePath string
var log zerolog.Logger

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Trace().Msgf("[api] %s %s %s", r.Method, r.URL, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func middlewareAuth(username, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.RemoteAddr, "127.") && !strings.HasPrefix(r.RemoteAddr, "[::1]") && r.RemoteAddr != "@" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != username || pass != password {
				w.Header().Set("Www-Authenticate", `Basic realm="go2rtc"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		next.ServeHTTP(w, r)
	})
}

var mu sync.Mutex

func apiHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if app.Info["stats"] == nil {
		app.Info["stats"] = make(map[string]interface{})
	}

	cpuUsage, err := core.GetCPUUsage(0)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to get CPU usage."}`))
		log.Warn().Err(err).Msg("[api] cpu stat")
		return
	}

	memUsage, err := core.GetRAMUsage()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to get memory usage."}`))
		log.Warn().Err(err).Msg("[api] ram stat")
		return
	}

	hostInfo, err := core.GetHostInfo()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to get system info"}`))
		log.Warn().Err(err).Msg("[api] system info")
		return
	}

	app.Info["stats"].(map[string]interface{})["cpu"] = cpuUsage
	app.Info["stats"].(map[string]interface{})["mem"] = memUsage
	app.Info["system"] = hostInfo
	app.Info["version"] = app.Version

	app.Info["host"] = r.Host
	app.Info["ffmpeg"] = app.FFmpegVersion

	ResponseJSON(w, app.Info)
}

func exitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	s := r.URL.Query().Get("code")
	code, err := strconv.Atoi(s)

	// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_08_02
	if err != nil || code < 0 || code > 125 {
		http.Error(w, "Code must be in the range [0, 125]", http.StatusBadRequest)
		return
	}

	os.Exit(code)
}

func restartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	path, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug().Msgf("[api] restart %s", path)

	go syscall.Exec(path, os.Args, os.Environ())
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		levelFilter := r.URL.Query().Get("level")

		w.Header().Set("Content-Type", "application/jsonlines")

		// Assuming app.MemoryLog is a bytes.Buffer or similar
		scanner := bufio.NewScanner(bytes.NewReader(app.MemoryLog.Bytes()))
		var filteredLog bytes.Buffer

		for scanner.Scan() {
			var logEntry map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
				http.Error(w, "Error processing log entries", http.StatusInternalServerError)
				continue
			}

			// Filter by level if parameter is set
			if levelFilter == "" || logEntry["level"] == levelFilter {
				filteredLog.Write(scanner.Bytes())
				filteredLog.WriteByte('\n')
			}
		}

		if err := scanner.Err(); err != nil {
			http.Error(w, "Error reading log entries", http.StatusInternalServerError)
			return
		}

		_, _ = filteredLog.WriteTo(w)
	case "DELETE":
		if err := app.MemoryLog.Reset(); err != nil {
			log.Printf("Error resetting memory log: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		Response(w, "OK", "text/plain") // Assuming Response() correctly sets the status code to 204 or similar.
	default:
		w.Header().Set("Allow", "GET, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type Source struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Info     string `json:"info,omitempty"`
	URL      string `json:"url,omitempty"`
	Location string `json:"location,omitempty"`
}

func ResponseSources(w http.ResponseWriter, sources []*Source) {
	if len(sources) == 0 {
		http.Error(w, "no sources", http.StatusNotFound)
		return
	}

	var response = struct {
		Sources []*Source `json:"sources"`
	}{
		Sources: sources,
	}
	ResponseJSON(w, response)
}

func Error(w http.ResponseWriter, err error) {
	log.Error().Err(err).Caller(1).Send()

	http.Error(w, err.Error(), http.StatusInsufficientStorage)
}
