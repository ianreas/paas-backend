package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	//"bytes"
	"encoding/base64"
	"paas-backend/internal/services"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

// github token: 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

type BuildAndPushRequest struct {
	RepoFullName string `json:"repoFullName"`
	AccessToken  string `json:"accessToken"`
}

func BuildAndPushToECR(w http.ResponseWriter, r *http.Request) {
	// this is just declaring a variable
	var req BuildAndPushRequest

	// json.NewDecoder(r.Body).Decode(&req) => this part is decoding the request body into the req variable, like
	// const req: BuildAndPushRequest = response.data; in typescript.
	// we pass a pointer &req into Decode() so that allows the Decode() function to modify the req variable directly.
	// The error checking (if err := ... ; err != nil):
	// This is similar to a try-catch block in TypeScript. It's checking if there was an error during the JSON parsing.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// this is like res.status(400).send('Invalid request body'); (sending error response)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		// if there was an error, we return from the function early
		return
	}

	// Clone repository
	// we do this by creating a command and then running using exec.Run()
	// its also wrapped into the try catch block like the rest of the code using if err := ... ; err != nil
	repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(req.RepoFullName, "/", "_"))

	// Ensure the directory is removed before cloning
	if err := os.RemoveAll(repoDir); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove existing directory: %v", err), http.StatusInternalServerError)
		return
	}

	cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", req.AccessToken, req.RepoFullName), repoDir)
	// if err := cloneCmd.Run(); err != nil {
	// 	http.Error(w, fmt.Sprintf("Failed to clone repository: %v", err), http.StatusInternalServerError)
	// 	return
	// }
	// defer os.RemoveAll(repoDir) // Clean up after we're done
	var out, errOut strings.Builder
	cloneCmd.Stdout = &out
	cloneCmd.Stderr = &errOut

	if err := cloneCmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to clone repository: %v. Output: %s. Error: %s", err, out.String(), errOut.String()), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(repoDir) // Clean up after we're done

	// Find Dockerfile
	dockerfilePath, err := services.FindDockerfile(repoDir)
	if err != nil {
		http.Error(w, "Dockerfile not found", http.StatusBadRequest)
		return
	}

	// Build Docker image
	imageName := filepath.Base(req.RepoFullName)
	buildCmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", fmt.Sprintf("%s:latest", imageName), filepath.Dir(dockerfilePath))
	if err := buildCmd.Run(); err != nil {
		http.Error(w, "Failed to build Docker image", http.StatusInternalServerError)
		return
	}

	//Push to ECR
	if err := pushToECR(r.Context(), imageName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to push to ECR: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Image built and pushed successfully"})
}

func pushToECR(ctx context.Context, imageName string) error {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create ECR client
	client := ecr.NewFromConfig(cfg)

	// Get ECR authorization token
	authOutput, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return fmt.Errorf("failed to get ECR auth token: %w", err)
	}

	// Decode auth token and extract username/password
	authToken, err := base64.StdEncoding.DecodeString(*authOutput.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return fmt.Errorf("failed to decode auth token: %w", err)
	}
	parts := strings.SplitN(string(authToken), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid auth token format")
	}
	username, password := parts[0], parts[1]

	// Login to ECR
	loginCmd := exec.Command("docker", "login", "--username", username, "--password-stdin", "590183673953.dkr.ecr.us-east-1.amazonaws.com")
	loginCmd.Stdin = strings.NewReader(password)
	loginOut, err := loginCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to login to ECR: %w, output: %s", err, loginOut)
	}

	// Tag the image
	ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app:%s", imageName)
	tagCmd := exec.Command("docker", "tag", fmt.Sprintf("%s:latest", imageName), ecrImageName)
	tagOut, err := tagCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to tag image: %w, output: %s", err, tagOut)
	}

	// Push the image
	pushCmd := exec.Command("docker", "push", ecrImageName)
	pushOut, err := pushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push image to ECR: %w, output: %s", err, pushOut)
	}

	return nil
}

