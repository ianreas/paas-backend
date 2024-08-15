// services/docker_service.go
package services

import (
    "fmt"
    "os/exec"
    "path/filepath"
)

type DockerServiceImpl struct{}

func NewDockerService() DockerService {
    return &DockerServiceImpl{}
}

func (s *DockerServiceImpl) BuildImage(dockerfilePath, imageName string) error {
    buildCmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", fmt.Sprintf("%s:latest", imageName), filepath.Dir(dockerfilePath))
    return buildCmd.Run()
}