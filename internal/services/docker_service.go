package services

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindDockerfile(dir string) (string, error) {
	var dockerfilePath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "Dockerfile" {
			dockerfilePath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if dockerfilePath == "" {
		return "", fmt.Errorf("Dockerfile not found in %s", dir)
	}
	return dockerfilePath, nil
}