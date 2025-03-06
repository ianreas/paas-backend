// services/docker_service.go
package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/client"

	"log"

	"bufio"
)

type DockerServiceImpl struct {
    client *client.Client
}

func NewDockerService() (DockerService, error) {
    cli, err := client.NewClientWithOpts(
        client.FromEnv,
        client.WithAPIVersionNegotiation(),
        client.WithVersion("1.41"), // Use a specific API version
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }

    // Test Docker connection
    ctx := context.Background()
    ping, err := cli.Ping(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to ping Docker daemon: %w", err)
    }
    log.Printf("Successfully connected to Docker daemon (API Version: %s)", ping.APIVersion)

    return &DockerServiceImpl{
        client: cli,
    }, nil
}

func (s *DockerServiceImpl) BuildImage(dockerfilePath, imageName string) error {
    dir := filepath.Dir(dockerfilePath)
    log.Printf("Building Docker image from context directory: %s", dir)
    log.Printf("Using Dockerfile: %s", filepath.Base(dockerfilePath))
    log.Printf("Target image name: %s", imageName)

    // List all files in the directory for debugging
    if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        rel, err := filepath.Rel(dir, path)
        if err != nil {
            return err
        }
        log.Printf("Found file in context: %s", rel)
        return nil
    }); err != nil {
        log.Printf("Warning: error listing directory contents: %v", err)
    }

    // Build the image using docker CLI
    buildCmd := exec.Command("docker", "build",
        "--no-cache",
        "--force-rm",
        "-t", imageName,
        "-f", dockerfilePath,
        "--platform", "linux/amd64",
        dir,
    )

    // Set up pipes for stdout and stderr
    stdout, err := buildCmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("failed to create stdout pipe: %w", err)
    }
    stderr, err := buildCmd.StderrPipe()
    if err != nil {
        return fmt.Errorf("failed to create stderr pipe: %w", err)
    }

    // Start the command
    if err := buildCmd.Start(); err != nil {
        return fmt.Errorf("failed to start build command: %w", err)
    }

    // Create a channel to signal when we're done reading output
    done := make(chan bool)

    // Read stdout in a goroutine
    go func() {
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            log.Printf("Build output: %s", scanner.Text())
        }
        done <- true
    }()

    // Read stderr in a goroutine
    go func() {
        scanner := bufio.NewScanner(stderr)
        for scanner.Scan() {
            log.Printf("Build error: %s", scanner.Text())
        }
        done <- true
    }()

    // Wait for both output readers to finish
    <-done
    <-done

    // Wait for the command to finish
    if err := buildCmd.Wait(); err != nil {
        return fmt.Errorf("build command failed: %w", err)
    }

    // Verify the image exists
    inspectCmd := exec.Command("docker", "inspect", imageName)
    if err := inspectCmd.Run(); err != nil {
        return fmt.Errorf("failed to verify built image: %w", err)
    }

    log.Printf("Successfully built image: %s", imageName)
    return nil
}