package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/pprof/profile"
)

const storagePath = "./"

func saveToFile(machineID string, file io.Reader) error {
	tempFile, err := ioutil.TempFile("", "profile_*.prof")
	if err != nil {
		return fmt.Errorf("could not create temp file: %v", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return fmt.Errorf("could not write to temp file: %v", err)
	}

	tempFile.Seek(0, io.SeekStart)
	prof, err := profile.Parse(tempFile)
	if err != nil {
		return fmt.Errorf("invalid profile format: %v", err)
	}

	profileFilePath := filepath.Join(storagePath, fmt.Sprintf("profile_%s.prof", machineID))

	var mergedProf *profile.Profile
	if _, err := os.Stat(profileFilePath); err == nil {
		existingFile, err := os.Open(profileFilePath)
		if err != nil {
			return fmt.Errorf("could not open existing profile: %v", err)
		}
		defer existingFile.Close()

		existingProf, err := profile.Parse(existingFile)
		if err != nil {
			return fmt.Errorf("invalid existing profile format: %v", err)
		}

		mergedProf, err = profile.Merge([]*profile.Profile{existingProf, prof})
		if err != nil {
			return fmt.Errorf("could not merge profiles: %v", err)
		}
	} else {
		mergedProf = prof
	}

	outputFile, err := os.Create(profileFilePath)
	if err != nil {
		return fmt.Errorf("could not create profile file: %v", err)
	}
	defer outputFile.Close()

	if err := mergedProf.Write(outputFile); err != nil {
		return fmt.Errorf("could not write merged profile: %v", err)
	}

	return nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	machineID := r.Header.Get("Machine-ID")
	if machineID == "" {
		http.Error(w, "Machine-ID header missing", http.StatusBadRequest)
		return
	}

	err := saveToFile(machineID, r.Body)
	if err != nil {
		log.Printf("failed to save profile: %v", err)
		http.Error(w, "could not save profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("profile from machine %s received successfully", machineID)
}
func main() {
	http.HandleFunc("/", Handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
