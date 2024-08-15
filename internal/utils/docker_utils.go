package utils

import (
    "os"
    "path/filepath"
    "fmt"
)

func FindDockerfile(dir string) (string, error) {
    var dockerfilePath string
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && (info.Name() == "Dockerfile" || info.Name() == "dockerfile") {
            dockerfilePath = path
            return filepath.SkipDir
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