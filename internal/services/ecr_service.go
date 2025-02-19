// services/ecr_service.go
package services

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
    "paas-backend/internal/utils"

     "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// this creates an instance of an ECRService. 
// and returns a pointer to it. 
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

    // Generate ECR-compatible image name FIRST
    timestamp := time.Now().Format("20060102150405")
    ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/%s:%s", 
        filepath.Base(repoFullName), // Extracts "dummy-express-eks" from "ianreas/dummy-express-eks"
        timestamp)

    // Get ECR auth before building
    ecrAuth, err := s.ecrRepo.GetAuthToken(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get ECR credentials: %w", err)
    }

    // Pass auth to build
    if err := s.dockerService.BuildImage(dockerfilePath, ecrImageName, ecrAuth); err != nil {
        return "", fmt.Errorf("failed to build Docker image: %w", err)
    }

    // Push using the same ECR name
    if _, err := s.ecrRepo.PushImage(ctx, ecrImageName); err != nil {
        return "", fmt.Errorf("failed to push to ECR: %w", err)
    }

    return ecrImageName, nil
}

// func (s *ECRServiceImpl) cloneRepository(repoFullName, accessToken string) (string, error) {
//     repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(repoFullName, "/", "_"))
//     if err := os.RemoveAll(repoDir); err != nil {
//         return "", fmt.Errorf("failed to remove existing directory: %w", err)
//     }

//     cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", accessToken, repoFullName), repoDir)
//     var out, errOut strings.Builder
//     cloneCmd.Stdout = &out
//     cloneCmd.Stderr = &errOut

//     if err := cloneCmd.Run(); err != nil {
//         return "", fmt.Errorf("failed to clone repository: %w. Output: %s. Error: %s", err, out.String(), errOut.String())
//     }

//     return repoDir, nil
// }
// func (s *ECRServiceImpl) cloneRepository(repoFullName, accessToken string) (string, error) {
//     // Check if git is available
//     if _, err := exec.LookPath("git"); err != nil {
//         return "", fmt.Errorf("git is not installed: %w", err)
//     }

//     repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(repoFullName, "/", "_"))
//     if err := os.RemoveAll(repoDir); err != nil {
//         return "", fmt.Errorf("failed to remove existing directory: %w", err)
//     }

//     // Create the directory
//     if err := os.MkdirAll(repoDir, 0755); err != nil {
//         return "", fmt.Errorf("failed to create directory: %w", err)
//     }

//     // Construct the clone URL
//     cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", accessToken, repoFullName)

//     // Set up the command with working directory
//     cloneCmd := exec.Command("git", "clone", cloneURL, ".")
//     cloneCmd.Dir = repoDir

//     // Set up output capturing
//     var out, errOut strings.Builder
//     cloneCmd.Stdout = &out
//     cloneCmd.Stderr = &errOut

//     // Set up environment variables
//     cloneCmd.Env = append(os.Environ(),
//         "GIT_TERMINAL_PROMPT=0",
//         "GIT_SSL_NO_VERIFY=true",
//     )

//     // Run the command
//     if err := cloneCmd.Run(); err != nil {
//         // Clean up the directory in case of failure
//         os.RemoveAll(repoDir)
//         return "", fmt.Errorf("failed to clone repository: %w\nOutput: %s\nError: %s", 
//             err, out.String(), errOut.String())
//     }

//     return repoDir, nil
// }

func (s *ECRServiceImpl) cloneRepository(repoFullName, accessToken string) (string, error) {
    repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(repoFullName, "/", "_"))
    if err := os.RemoveAll(repoDir); err != nil {
        return "", fmt.Errorf("failed to remove existing directory: %w", err)
    }

    // Create the directory
    if err := os.MkdirAll(repoDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create directory: %w", err)
    }

    // Construct the clone URL
    cloneURL := fmt.Sprintf("https://github.com/%s.git", repoFullName)

    // Set up authentication
    auth := &http.BasicAuth{
        Username: "x-access-token", // This can be anything except empty string
        Password: accessToken,
    }

    // Clone the repository
    _, err := git.PlainClone(repoDir, false, &git.CloneOptions{
        URL:      cloneURL,
        Auth:     auth,
        Progress: os.Stdout, // This will show clone progress
    })
    if err != nil {
        // Clean up the directory in case of failure
        os.RemoveAll(repoDir)
        return "", fmt.Errorf("failed to clone repository: %w", err)
    }

    return repoDir, nil
}