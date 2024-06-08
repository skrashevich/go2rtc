package app

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"syscall"
)

var (
	Version       string
	UserAgent     string
	ConfigPath    string
	FFmpegVersion string
	Info          = make(map[string]any)
)

/*const usage = `Usage of go2rtc:
	Version    string
	UserAgent  string
	ConfigPath string
	FFmpegVersion string
	Info       = make(map[string]any)
)

const usage = `Usage of go2rtc:

  -c, --config   Path to config file or config string as YAML or JSON, support multiple
  -d, --daemon   Run in background
  -v, --version  Print version and exit
`*/

func Init() {
	var config flagConfig
	var daemon bool
	// configflag := false
	var version bool

	flag.Var(&config, "config", "")
	flag.Var(&config, "c", "")
	flag.BoolVar(&daemon, "daemon", false, "")
	flag.BoolVar(&daemon, "d", false, "")
	flag.BoolVar(&version, "version", false, "")
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

	if os.Getppid() == 1 || syscall.Getppid() == 1 {
		daemon = false
	} else {
		parent, err := os.FindProcess(os.Getppid())
		if err != nil || parent.Pid < 1 {
			daemon = false
		}
	}

	if daemon {
		if runtime.GOOS == "windows" {
			fmt.Println("Daemon not supported on Windows")
			os.Exit(1)
		}

		// Re-run the program in background and exit
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		if err := cmd.Start(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Running in daemon mode with PID:", cmd.Process.Pid)
		os.Exit(0)
	}

	UserAgent = "go2rtc/" + Version

	initConfig(config)
	initLogger()

	revision, vcsTime := readRevisionTime()

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	Logger.Info().Str("version", Version).Str("platform", platform).Str("revision", revision).Msg("go2rtc")
	Logger.Debug().Str("version", runtime.Version()).Str("vcs.time", vcsTime).Msg("build")

	if ConfigPath != "" {
		Logger.Info().Str("path", ConfigPath).Msg("config")
	}
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
func GetVersionString() string {
	revision, vcsTime := readRevisionTime()

	return fmt.Sprintf("%s%s: %s %s/%s", Version, revision, vcsTime, runtime.GOOS, runtime.GOARCH)
}
