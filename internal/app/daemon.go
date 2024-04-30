//go:build !windows

package app

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
)

func runDaemon() {
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
