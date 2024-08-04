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

// func pushToECR(ctx context.Context, imageName string) error {
// 	// Load AWS configuration
// 	cfg, err := config.LoadDefaultConfig(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to load AWS config: %w", err)
// 	}

// 	// Create ECR client
// 	client := ecr.NewFromConfig(cfg)

// 	// Get ECR authorization token
// 	authOutput, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
// 	if err != nil {
// 		return fmt.Errorf("failed to get ECR auth token: %w", err)
// 	}

// 	fmt.Print(authOutput)

// 	// Use the auth token to login to ECR
// 	// Note: In a real implementation, you'd need to decode the auth token and use it for Docker login
// 	// This is a simplified example
// 	// this command worked in the cli
// 	// 1)  aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app
// 	//loginCmd := exec.Command("docker", "login", "-u", "AWS", "-p", "password", "590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app")
// 	// loginCmd := exec.Command("aws", "ect", "get-login-password", "--region", "us-east-1", "|", "docker", "login", "--username", "AWS", "--password-stdin", "590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app")
// 	// var pushToEcrOut, pushToECRErr strings.Builder
// 	// loginCmd.Stdout = &pushToEcrOut
// 	// loginCmd.Stderr = &pushToECRErr
// 	// if err := loginCmd.Run(); err != nil {
// 	// 	return fmt.Errorf("failed to login to ECR: %v. Output: %s, Error: %s", err, pushToEcrOut.String(), pushToECRErr.String())
// 	// }
// 	//590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app

// 	//eyJwYXlsb2FkIjoidWR4RzNRdTQvV2RMdUxkWThwNUdNR2UrSUVYQ0ZFMnFiSS9GNnY5V29RUGNXc1NvZ2RoRHJWMm00emhValJHRUJ2TDQ5NFh3MmFoOTlqVGwrVnNUd2FpMk5FWWI5S2FUUnNrTWpEY2V2eDY5KzFWeGU3bWh2Zlp4NnlkQUhMZ3JKVHp4VkZTQndpVE9OYzBpV2k2dUNUR29RcURHdWF3REhXTm1yWGdTRHFKZnRudkMrYTU5M1BIV2Z4MEdGZU5uMWVZU0VDNDZCUGIwUXlua3k1TnhMVlJQU2VDVDlGMVZNSVI4MmNBT0k2clZhNG95cW5COWFuMFpPWnV2Z0o5RUZOKzIzV3JGQzZRSldWQlFmcjJSRldpbkZHRjhENjcyNURwTTU4OEhrQ0I2M3RUbEIrcDc5NTl6b2hvYWgrdHRNa2lzNE1sWm9KTzQvdGdUSkNhMi9lWE5ZUlVFVXFkVkhIUTc4Qjcra2daeThOa0w3em5TUGtWTjBrRlB0bGIvMTdLZWoyM1JYMkJXMUZlamoyQk1aQ1h6aksxL014UG16N1pkbzFndzRpNlI1TzJITUFCREpBc0xsbWpiZnVySkR6Mk1DdDZaS0wzRVdYM0l5bGRzZHIyemJpTUx5b0NUd0NsSzB1cm9kTTRMS1EySFNFWDJyY0xOc01FTjVNMU51a0pRQmhNa0QreHpXWEdZYXlpTU04SDRMb1B2MGQvYWtJemc4ZDliYjV4Z3pkcllscXRjQzdxaWU5YU9qODlDeDkwNUhWdEVNQWNacVBlOEo2Q0pINE5zWDNrRXlJbldxRkhBL3NtVjdLQkpBaUdtaEFmcUYrVU0zUUh0SVVOdEtucUJrcmZOVGduaU5lTzVRRkllNWEvL3VtcmZTcXB4YkU4MnhhWWg1ajhacVppWWlla0UxK1lpOWYwTlRLMGJld2EvY3IzZnAzT1IxZ3Y3TVEyWjNoOFkxTnYrRkdlWlBIOVd0VDBKa3ppOVJ0c0pvOWlWd1JDV0xtRGV3YXBKNmw5SEwrbUpxQWdDL05LMGlkOGptdnRSZjgwZThZQXBIMDUvV0V5RmxWUXlEYmhhemRzbmZLR1p3VEh3dXNPVTl1ZnFaNEZXWkYyUWZNSEE2WjJHZ1RXUkltV2dpeFdTSEo2RlY3a2hxQUtCRVR0RzI2RktwVm43SVp0ZkFkbWFNdDVGZ0FQS0hWK2NGK0luVlVBczg0c09FS3pJa1IyUHcvemJCMUlQdHdLRGJCUXdsbS9vaXNOeGJ6N05LckR1WmRLK1lneFV2UndlcmxkeWFkRzlXc0xhOTgwcWlmeXZ4Zz09IiwiZGF0YWtleSI6IkFRRUJBSGh3bTBZYUlTSmVSdEptNW4xRzZ1cWVla1h1b1hYUGU1VUZjZTlScTgvMTR3QUFBSDR3ZkFZSktvWklodmNOQVFjR29HOHdiUUlCQURCb0Jna3Foa2lHOXcwQkJ3RXdIZ1lKWUlaSUFXVURCQUV1TUJFRURES1p1K0xOdW53YXFRcmZMQUlCRUlBN3hFRjF3NmJlTWdWR0d0VS92VFZSeFRSaEdQeE9IcldjaVJsek1FVzJIc1BuaWlGVFp4RVhDUnBFVHZob1NML3ZTT3doRXFZU0pCQUpFV289IiwidmVyc2lvbiI6IjIiLCJ0eXBlIjoiREFUQV9LRVkiLCJleHBpcmF0aW9uIjoxNzIyNzUyNTg3fQ==

// 	// First command: aws ecr get-login-password --region us-east-1
//     getPasswordCmd := exec.Command("aws", "ecr", "get-login-password", "--region", "us-east-1")

//     // Buffer to capture the output of the first command
//     var outBuf, errBuf bytes.Buffer
//     getPasswordCmd.Stdout = &outBuf
//     getPasswordCmd.Stderr = &errBuf

//     // Run the first command
//     if err := getPasswordCmd.Run(); err != nil {
//         return fmt.Errorf("failed to get login password: %v. Output: %s, Error: %s", err, outBuf.String(), errBuf.String())
//     }

//     // Second command: docker login --username AWS --password-stdin <repoURL>
//     dockerLoginCmd := exec.Command("docker", "login", "--username", "AWS", "--password-stdin", "590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app")

//     // Use the output of the first command as the input for the second command
//     dockerLoginCmd.Stdin = &outBuf

//     // Buffer to capture the output of the second command
//     var loginOutBuf, loginErrBuf bytes.Buffer
//     dockerLoginCmd.Stdout = &loginOutBuf
//     dockerLoginCmd.Stderr = &loginErrBuf

//     // Run the second command
//     if err := dockerLoginCmd.Run(); err != nil {
//         return fmt.Errorf("failed to login to ECR: %v. Output: %s, Error: %s", err, loginOutBuf.String(), loginErrBuf.String())
//     }

// 	// Tag the image for ECR
// 	ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app/%s", imageName)
// 	tagCmd := exec.Command("docker", "tag", imageName, ecrImageName)
// 	if err := tagCmd.Run(); err != nil {
// 		return fmt.Errorf("failed to tag image: %w", err)
// 	}

// 	// Push the image to ECR
// 	pushCmd := exec.Command("docker", "push", ecrImageName)
// 	var pushOut, pushErr strings.Builder
// 	pushCmd.Stdout = &pushOut
// 	pushCmd.Stderr = &pushErr
// 	if err := pushCmd.Run(); err != nil {
// 		//return fmt.Errorf("failed to push image to ECR: %w", err)
// 		return fmt.Errorf("failed to push image to ECR: %v. Output: %s, Error: %s", err, pushOut.String(), pushErr.String())
// 	}

// 	return nil
// }
