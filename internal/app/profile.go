package app

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"runtime/pprof"
	"time"
)

var UploadURL string

func generateMachineID() string {
	initLogger()
	id := make([]byte, 16)
	_, err := rand.Read(id)
	if err != nil {
		Logger.Fatal().Err(err).Msg("could not generate machine ID")
	}
	return hex.EncodeToString(id)
}

func StartProfiling(machineID string) {
	if machineID == "" {
		machineID = generateMachineID()
	}
	if UploadURL == "" {
		UploadURL = "https://functions.yandexcloud.net/d4eilb6lq8elq0ritcv7"
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			profileFilename := "heap.prof"
			f, err := os.Create(profileFilename)
			if err != nil {
				Logger.Fatal().Err(err).Msg("could not create profile file")
			}

			if err := pprof.WriteHeapProfile(f); err != nil {
				Logger.Fatal().Err(err).Msg("could not write heap profile")
			}
			f.Close()

			uploadProfile(profileFilename, machineID)
		}
	}
}

func uploadProfile(filename, machineID string) {
	file, err := os.Open(filename)
	if err != nil {
		Logger.Fatal().Err(err).Msg("could not open profile file")
	}
	defer file.Close()

	request, err := http.NewRequest("POST", UploadURL, file)
	if err != nil {
		Logger.Fatal().Err(err).Msg("could not create upload request")
	}

	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Machine-ID", machineID)

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		Logger.Fatal().Err(err).Msg("could not upload profile")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		Logger.Fatal().Msgf("upload failed with status: %v", resp.Status)
	}
	Logger.Info().Msg("profile uploaded successfully")
}
