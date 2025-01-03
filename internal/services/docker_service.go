// services/docker_service.go
package services

import (
    "fmt"
    "context"
    "io"
    "os"
    "path/filepath"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/client"
    
    "github.com/docker/docker/pkg/archive"
)

type DockerServiceImpl struct {
    client *client.Client
}

func NewDockerService() (DockerService, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }
    return &DockerServiceImpl{
        client: cli,
    }, nil
}

func (s *DockerServiceImpl) BuildImage(dockerfilePath, imageName string) error {
    ctx := context.Background()
    
    // Create build context
    dir := filepath.Dir(dockerfilePath)
    tar, err := archive.TarWithOptions(dir, &archive.TarOptions{})
    if err != nil {
        return fmt.Errorf("failed to create build context: %w", err)
    }
    defer tar.Close()

    // Build options
    opts := types.ImageBuildOptions{
        Dockerfile: filepath.Base(dockerfilePath),
        Tags:      []string{imageName},
        Platform:  "linux/amd64",
        NoCache:   true,
        Remove:    true,
    }

    // Build the image
    resp, err := s.client.ImageBuild(ctx, tar, opts)
    if err != nil {
        return fmt.Errorf("failed to build image: %w", err)
    }
    defer resp.Body.Close()

    // Read the response
    _, err = io.Copy(os.Stdout, resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read build response: %w", err)
    }

    return nil
}