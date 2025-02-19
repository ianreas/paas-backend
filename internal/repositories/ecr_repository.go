package repositories

import (
    "context"
    "encoding/base64"
    "fmt"
    "os/exec"
    "strings"
    "log"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ecr"
)

type ECRRepository struct {
    client *ecr.Client
}

type ECRAuth struct {
    Username string
    Password string
    Registry string
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
    //repoName := parts[0]
    tag := parts[1]

    ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app:%s", tag)

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

func (r *ECRRepository) GetAuthToken(ctx context.Context) (*ECRAuth, error) {
    authOutput, err := r.client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
    if err != nil {
        return nil, fmt.Errorf("failed to get ECR auth token: %w", err)
    }

    authToken, err := base64.StdEncoding.DecodeString(*authOutput.AuthorizationData[0].AuthorizationToken)
    if err != nil {
        return nil, fmt.Errorf("failed to decode auth token: %w", err)
    }
    
    parts := strings.SplitN(string(authToken), ":", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid auth token format")
    }

    return &ECRAuth{
        Username: parts[0],
        Password: parts[1],
        Registry: "590183673953.dkr.ecr.us-east-1.amazonaws.com",
    }, nil
}