package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/AlexxIT/go2rtc/pkg/shell"
	"github.com/AlexxIT/go2rtc/pkg/yaml"
	"github.com/rs/zerolog/log"
)

var (
	Version               = "1.9.2.2"
	UserAgent             = "go2rtc/" + Version
	FFmpegVersion         = ""
	DefaultConfigFileName = "go2rtc.yaml"
)

var (
	ConfigPath string
	Info       = map[string]any{
		"version": Version,
	}
)

/*const usage = `Usage of go2rtc:

  -c, --config   Path to config file or config string as YAML or JSON, support multiple
  -d, --daemon   Run in background
  -v, --version  Print version and exit
`*/

func Init() {
	var confs Config
	var daemon bool
	// configflag := false
	var version bool

	flag.Var(&confs, "config", "go2rtc config (path to file or raw text), support multiple")
	flag.Var(&confs, "c", "")
	if runtime.GOOS != "windows" {
		flag.BoolVar(&daemon, "daemon", false, "Run program in background")
	}
	flag.BoolVar(&version, "version", false, "Print the version of the application and exit")
	flag.BoolVar(&version, "v", false, "")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of go2rtc\nversion %s\n\n", GetVersionString())
		flag.VisitAll(func(f *flag.Flag) {
			pname := ""
			if f.Usage != "" {
				switch f.Name {
				case "config":
					pname = "-c --config"
					break
				case "daemon":
					pname = "-d --daemon"
					break
				case "version":
					pname = "-v --version"
					break
				default:
					pname = "-" + f.Name
				}
				fmt.Fprintf(os.Stderr, "\t%s\n\t\t%s (default %q)\n", pname, f.Usage, f.DefValue)
			}
		})
		fmt.Fprintf(os.Stderr, "\t%s\n\t\t%s\n", "-h --help", "Print this help")
	}

	flag.Parse()

	if version {
		fmt.Println("go2rtc version " + GetVersionString())
		os.Exit(0)
	}

	if daemon {
		if runtime.GOOS == "windows" {
			fmt.Println("Daemon not supported on Windows")
			os.Exit(1)
		}

		args := os.Args[1:]
		for i, arg := range args {
			if arg == "-daemon" {
				args[i] = ""
			}
		}
		// Re-run the program in background and exit
		cmd := exec.Command(os.Args[0], args...)
		if err := cmd.Start(); err != nil {
			log.Fatal().Err(err).Send()
		}
		fmt.Println("Running in daemon mode with PID:", cmd.Process.Pid)
		os.Exit(0)
	}

	if confs == nil {
		confs = []string{DefaultConfigFileName}
	}

	for _, conf := range confs {
		if len(conf) < 1 {
			continue
		}
		if conf[0] == '{' {
			// config as raw YAML or JSON
			configs = append(configs, []byte(conf))
		} else if data := parseConfString(conf); data != nil {
			configs = append(configs, data)
		} else {
			// config as file
			if ConfigPath == "" {
				ConfigPath = conf
			}

			if data, _ = os.ReadFile(conf); data == nil {
				continue
			}

			data = []byte(shell.ReplaceEnvVars(string(data)))
			configs = append(configs, data)
		}
	}

	/*	if !configflag {
		data, _ := os.ReadFile(DefaultConfigFileName)
		if data != nil {
			data = []byte(shell.ReplaceEnvVars(string(data)))
			configs = prepend(configs, data)
			ConfigPath = DefaultConfigFileName
		}
	}*/

	if ConfigPath != "" {
		if !filepath.IsAbs(ConfigPath) {
			if cwd, err := os.Getwd(); err == nil {
				ConfigPath = filepath.Join(cwd, ConfigPath)
			}
		}
		Info["config_path"] = ConfigPath
	}
	var cfg struct {
		Mod map[string]string `yaml:"log"`
	}

	LoadConfig(&cfg)

	log.Logger = NewLogger(cfg.Mod["format"], cfg.Mod["level"])

	modules = cfg.Mod

	revision, vcsTime := readRevisionTime()

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	log.Info().Str("version", Version).Str("platform", platform).Str("revision", revision).Msg("go2rtc")
	log.Debug().Str("version", runtime.Version()).Str("vcs.time", vcsTime).Msg("build")

	if ConfigPath != "" {
		log.Info().Str("path", ConfigPath).Msg("config")
	}

	migrateStore()
}

// prepend adds an item to the beginning of a slice. It works with slices of any type,
// thanks to Go's type parameters feature. The function creates a new slice with enough
// capacity to hold the additional item plus all existing items in the input slice.
// It then appends the new item followed by all items of the input slice to this new slice.
//
// Parameters:
//   - slice: The original slice to which the item will be prepended.
//   - item: The item to prepend to the slice.
//
// Returns:
//
//	A new slice with the item added at the beginning.
func prepend[T any](slice []T, item T) []T {
	result := make([]T, 0, len(slice)+1)
	result = append(result, item)
	result = append(result, slice...)
	return result
}

func LoadConfig(v any) {
	for _, data := range configs {
		if err := yaml.Unmarshal(data, v); err != nil {
			log.Warn().Err(err).Msg("[app] read config")
		}
	}
}

func PatchConfig(key string, value any, path ...string) error {
	if ConfigPath == "" {
		return errors.New("config file disabled")
	}

	// empty config is OK
	b, _ := os.ReadFile(ConfigPath)

	b, err := yaml.Patch(b, key, value, path...)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath, b, 0o644)
}

// internal

type Config []string

func (c *Config) String() string {
	return strings.Join(*c, " ")
}

func (c *Config) Set(value string) error {
	*c = append(*c, value)
	return nil
}

var configs [][]byte

func GetVersionString() string {
	revision, vcsTime := readRevisionTime()

	return fmt.Sprintf("%s%s: %s %s/%s", Version, revision, vcsTime, runtime.GOOS, runtime.GOARCH)
}

func readRevisionTime() (revision, vcsTime string) {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					revision = setting.Value[:7]
				} else {
					revision = setting.Value
				}
			case "vcs.time":
				vcsTime = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					revision = "mod." + revision
				}
			}
		}
	}
	return
}

func parseConfString(s string) []byte {
	i := strings.IndexByte(s, '=')
	if i < 0 {
		return nil
	}

	items := strings.Split(s[:i], ".")
	if len(items) < 2 {
		return nil
	}

	// `log.level=trace` => `{log: {level: trace}}`
	var pre string
	suf := s[i+1:]
	for _, item := range items {
		pre += "{" + item + ": "
		suf += "}"
	}

	return []byte(pre + suf)
}
