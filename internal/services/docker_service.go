// services/docker_service.go
package services

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"paas-backend/internal/repositories"
	"strings"
        "github.com/docker/docker/api/types/registry"

	"github.com/docker/docker/pkg/archive"
)

type DockerServiceImpl struct {
    client *client.Client
}

type dockerBuildLine struct {
    Stream      string `json:"stream"`
    Error       string `json:"error"`
    ErrorDetail struct {
        Message string `json:"message"`
    } `json:"errorDetail"`
}

func NewDockerService() (DockerService, error) {
    cli, err := client.NewClientWithOpts(
        client.WithHost("unix:///var/run/docker.sock"),
        client.WithAPIVersionNegotiation(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }
    return &DockerServiceImpl{
        client: cli,
    }, nil
}

func (s *DockerServiceImpl) BuildImage(dockerfilePath, imageName string, ecrAuth *repositories.ECRAuth) error {
    ctx := context.Background()
    
    // Create build context with explicit exclusion of node_modules
    dir := filepath.Dir(dockerfilePath)
    tar, err := archive.TarWithOptions(dir, &archive.TarOptions{
        ExcludePatterns: []string{"node_modules", ".git"},
    })
    if err != nil {
        return fmt.Errorf("failed to create build context: %w", err)
    }
    defer tar.Close()

    // Force pull base image and enable buildkit
    opts := types.ImageBuildOptions{
        Dockerfile:  filepath.Base(dockerfilePath),
        Tags:       []string{imageName},
        Platform:   "linux/amd64",
        NoCache:    false, // Use caching but with forced base image pull
        PullParent: true,  // Force pull base image
        Remove:     true,
        Version:    types.BuilderBuildKit, // Use BuildKit
    }

    // Add registry auth configuration
    opts.AuthConfigs = make(map[string]types.AuthConfig)
    if ecrAuth != nil {
        opts.AuthConfigs[ecrAuth.Registry] = types.AuthConfig{
            Username: ecrAuth.Username,
            Password: ecrAuth.Password,
        }
    }

    // Keep existing login command as fallback
    if ecrAuth != nil {
        loginCmd := exec.Command("docker", "login", "-u", ecrAuth.Username, "-p", ecrAuth.Password, ecrAuth.Registry)
        if output, err := loginCmd.CombinedOutput(); err != nil {
            log.Printf("Warning: Docker login failed: %v\n%s", err, string(output))
        }
    }

    // Build with verbose output
    resp, err := s.client.ImageBuild(ctx, tar, opts)
    if err != nil {
        return fmt.Errorf("failed to initiate Docker build: %w", err)
    }
    defer resp.Body.Close()

    // Enhanced output parsing
    scanner := bufio.NewScanner(resp.Body)
    var buildOutput strings.Builder
    
    for scanner.Scan() {
        line := scanner.Text()
        buildOutput.WriteString(line + "\n")
        
        var buildLog struct {
            Stream string `json:"stream"`
            Error  string `json:"error"`
        }
        if err := json.Unmarshal([]byte(line), &buildLog); err != nil {
            continue // Skip non-JSON lines
        }

        if buildLog.Error != "" {
            return fmt.Errorf("docker build error: %s", buildLog.Error)
        }
        
        // Validate base image layers early
        if strings.Contains(buildLog.Stream, "Pulling from library/node") {
            if strings.Contains(buildLog.Stream, "digest: sha256:") {
                expectedDigest := "29752c4f0657" // From your working image
                if !strings.Contains(buildLog.Stream, expectedDigest) {
                    return fmt.Errorf("base image digest mismatch: %s", buildLog.Stream)
                }
            }
        }
    }

    // Verify final image exists
    inspect, _, err := s.client.ImageInspectWithRaw(ctx, imageName)
    if err != nil {
        return fmt.Errorf("built image not found: %w\nBuild output:\n%s", 
            err, buildOutput.String())
    }

    if inspect.ID == "" {
        return fmt.Errorf("image build failed - no ID generated\nBuild output:\n%s", 
            buildOutput.String())
    }

    return nil
}