// services/ecr_service.go
package services

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "paas-backend/internal/utils"
)

func NewECRService(dockerService DockerService, ecrRepo ECRRepository, eksService EKSService) ECRService {
    return &ECRServiceImpl{
        dockerService: dockerService,
        ecrRepo:       ecrRepo,
        eksService:    eksService,
    }
}

type ECRServiceImpl struct {
    dockerService DockerService
    ecrRepo       ECRRepository
    eksService    EKSService
}

func (s *ECRServiceImpl) BuildAndPushToECR(ctx context.Context, repoFullName, accessToken string) (string, error) {
    repoDir, err := s.cloneRepository(repoFullName, accessToken)
    if err != nil {
        return "", fmt.Errorf("failed to clone repository: %w", err)
    }
    defer os.RemoveAll(repoDir)

    dockerfilePath, err := utils.FindDockerfile(repoDir)
    if err != nil {
        return "", fmt.Errorf("dockerfile not found: %w", err)
    }

    imageName := filepath.Base(repoFullName)
    if err := s.dockerService.BuildImage(dockerfilePath, imageName); err != nil {
        return "", fmt.Errorf("failed to build Docker image: %w", err)
    }

    ecrImageName, err := s.ecrRepo.PushImage(ctx, imageName)
    if err != nil {
        return "", fmt.Errorf("failed to push to ECR: %w", err)
    }

    if err := s.eksService.DeployToEKS(ctx, ecrImageName, imageName, 3000); err != nil {
        return "", fmt.Errorf("failed to deploy to EKS: %w", err)
    }

    return ecrImageName, nil
}

func (s *ECRServiceImpl) cloneRepository(repoFullName, accessToken string) (string, error) {
    repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(repoFullName, "/", "_"))
    if err := os.RemoveAll(repoDir); err != nil {
        return "", fmt.Errorf("failed to remove existing directory: %w", err)
    }

    cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", accessToken, repoFullName), repoDir)
    var out, errOut strings.Builder
    cloneCmd.Stdout = &out
    cloneCmd.Stderr = &errOut

    if err := cloneCmd.Run(); err != nil {
        return "", fmt.Errorf("failed to clone repository: %w. Output: %s. Error: %s", err, out.String(), errOut.String())
    }

    return repoDir, nil
}