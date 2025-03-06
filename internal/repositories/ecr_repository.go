package repositories

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type ECRRepository struct {
    client *ecr.Client
}


func NewECRRepository(ctx context.Context) (*ECRRepository, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := ecr.NewFromConfig(cfg)
    return &ECRRepository{client: client}, nil
}

func (r *ECRRepository) PushImage(ctx context.Context, imageName string) (string, error) {
    log.Println("Pushing image to ECR:", imageName)
    // Extract the repository name and tag from the imageName
    parts := strings.Split(imageName, ":")
    if len(parts) != 2 {
        return "", fmt.Errorf("invalid image name format: %s", imageName)
    }
    repoName := parts[0]
    tag := parts[1]

    // Ensure the repository exists
    if err := r.ensureRepositoryExists(ctx, repoName); err != nil {
        return "", fmt.Errorf("failed to ensure repository exists: %w", err)
    }

    // Format the ECR image name using the repository name
    ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/%s:%s", repoName, tag)

    authOutput, err := r.client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
    if err != nil {
        return "", fmt.Errorf("failed to get ECR auth token: %w", err)
    }

    authToken, err := base64.StdEncoding.DecodeString(*authOutput.AuthorizationData[0].AuthorizationToken)
    if err != nil {
        return "", fmt.Errorf("failed to decode auth token: %w", err)
    }
    parts = strings.SplitN(string(authToken), ":", 2)
    if len(parts) != 2 {
        return "", fmt.Errorf("invalid auth token format")
    }
    username, password := parts[0], parts[1]

    loginCmd := exec.Command("docker", "login", "--username", username, "--password-stdin", "590183673953.dkr.ecr.us-east-1.amazonaws.com")
    loginCmd.Stdin = strings.NewReader(password)
    if loginOut, err := loginCmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("failed to login to ECR: %w, output: %s", err, loginOut)
    }

    // Tag the local image with the ECR repository URI
    tagCmd := exec.Command("docker", "tag", imageName, ecrImageName)
    if tagOut, err := tagCmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("failed to tag image: %w, output: %s", err, tagOut)
    }

    pushCmd := exec.Command("docker", "push", ecrImageName)
    if pushOut, err := pushCmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("failed to push image to ECR: %w, output: %s", err, pushOut)
    }

    return ecrImageName, nil
}

func (r *ECRRepository) ensureRepositoryExists(ctx context.Context, repoName string) error {
    // Try to describe the repository first
    _, err := r.client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
        RepositoryNames: []string{repoName},
    })
    
    if err != nil {
        // If the repository doesn't exist, create it
        var rnf *types.RepositoryNotFoundException
        if errors.As(err, &rnf) {
            log.Printf("Repository %s not found, creating it...", repoName)
            _, err = r.client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
                RepositoryName: aws.String(repoName),
                ImageScanningConfiguration: &types.ImageScanningConfiguration{
                    ScanOnPush: true,
                },
            })
            if err != nil {
                return fmt.Errorf("failed to create repository: %w", err)
            }
            log.Printf("Repository %s created successfully", repoName)
            return nil
        }
        return fmt.Errorf("failed to describe repository: %w", err)
    }
    
    return nil
}