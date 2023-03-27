package app

import (
	"flag"
	"github.com/AlexxIT/go2rtc/pkg/shell"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

var Version = "1.3.1"
var UserAgent = "go2rtc/" + Version

var ConfigPath string
var Info = map[string]any{
	"version": Version,
}
var confs Config

func Init() {

	flag.Var(&confs, "config", "go2rtc config (path to file or raw text), support multiple")
	flag.Parse()

	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal().Msgf("could not create CPU profile: %s", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Msgf("could not start CPU profile: %s", err)
		}
		defer pprof.StopCPUProfile()
	}
	if confs == nil {
		confs = []string{"go2rtc.yaml"}
	}

	for _, conf := range confs {
		if conf[0] != '{' {
			// config as file
			if ConfigPath == "" {
				ConfigPath = conf
			}

			data, _ := os.ReadFile(conf)
			if data == nil {
				continue
			}

			data = []byte(shell.ReplaceEnvVars(string(data)))
			configs = append(configs, data)
		} else {
			// config as raw YAML
			configs = append(configs, []byte(conf))
		}
	}

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

	log.Info().Msgf("go2rtc version %s %s/%s", Version, runtime.GOOS, runtime.GOARCH)
}

func NewLogger(format string, level string) zerolog.Logger {
	var writer io.Writer = os.Stdout

	if format != "json" {
		writer = zerolog.ConsoleWriter{
			Out: writer, TimeFormat: "15:04:05.000",
			NoColor: writer != os.Stdout || format == "text",
		}
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano

	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(writer).With().Timestamp().Logger().Level(lvl)
}

func LoadConfig(v any) {
	for _, data := range configs {
		if err := yaml.Unmarshal(data, v); err != nil {
			log.Warn().Err(err).Msg("[app] read config")
		}
	}
}

func GetLogger(module string) zerolog.Logger {
	if s, ok := modules[module]; ok {
		lvl, err := zerolog.ParseLevel(s)
		if err == nil {
			return log.Level(lvl)
		}
		log.Warn().Err(err).Caller().Send()
	}

	return log.Logger
}

// ReloadConfig clears the current config and reloads it from the source(s)
func ReloadConfig() {
	configs = nil

	log.Info().Msg("Received SIGHUP, reloading configuration")

	for _, conf := range confs {
		if conf[0] != '{' {
			// config as file
			data, _ := os.ReadFile(conf)
			if data == nil {
				continue
			}

			data = []byte(shell.ReplaceEnvVars(string(data)))
			configs = append(configs, data)
		} else {
			// config as raw YAML
			configs = append(configs, []byte(conf))
		}
	}

	// Reload any specific configuration settings that need to be reloaded
	// after the config has been updated.
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

// modules log levels
var modules map[string]string
