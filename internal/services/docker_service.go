// services/docker_service.go
package services

import (
    "fmt"
    "os/exec"
    "path/filepath"
    "bytes"
)

type DockerServiceImpl struct{}

func NewDockerService() DockerService {
    return &DockerServiceImpl{}
}

func (s *DockerServiceImpl) BuildImage(dockerfilePath, imageName string) error {
    dir := filepath.Dir(dockerfilePath)
    buildCmd :=  exec.Command("docker", "build",
    "--platform", "linux/amd64",  // Explicitly set platform
    "-t", imageName,
    "-f", dockerfilePath,
    "--no-cache",  // Avoid caching issues
    dir)
    // Capture both stdout and stderr
    var stdout, stderr bytes.Buffer
    buildCmd.Stdout = &stdout
    buildCmd.Stderr = &stderr
    
    err := buildCmd.Run()
    if err != nil {
        // Combine stdout and stderr for a comprehensive error message
        return fmt.Errorf("docker build failed: %w\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
    }
    
    return nil
}