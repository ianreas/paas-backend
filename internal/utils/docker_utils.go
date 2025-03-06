package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func FindDockerfile(dir string) (string, error) {
    log.Printf("Searching for Dockerfile in directory: %s", dir)
    
    // First, check for Dockerfile in the root directory
    rootDockerfile := filepath.Join(dir, "Dockerfile")
    if info, err := os.Stat(rootDockerfile); err == nil && !info.IsDir() {
        log.Printf("Found Dockerfile at root: %s", rootDockerfile)
        return rootDockerfile, nil
    }

    // If not found in root, search recursively
    var dockerfilePath string
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            log.Printf("Error accessing path %s: %v", path, err)
            return err
        }
        
        if !info.IsDir() && (info.Name() == "Dockerfile" || info.Name() == "dockerfile") {
            dockerfilePath = path
            log.Printf("Found Dockerfile at: %s", path)
            return filepath.SkipDir
        }
        
        // Skip node_modules and .git directories
        if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".git") {
            return filepath.SkipDir
        }
        
        return nil
    })

    if err != nil {
        return "", fmt.Errorf("error searching for Dockerfile: %w", err)
    }
    
    if dockerfilePath == "" {
        // List the contents of the directory for debugging
        files, _ := os.ReadDir(dir)
        var fileList []string
        for _, file := range files {
            fileList = append(fileList, file.Name())
        }
        return "", fmt.Errorf("Dockerfile not found in %s. Directory contents: %v", dir, fileList)
    }

    return dockerfilePath, nil
}