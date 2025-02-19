package services

import (
	"context"
	"paas-backend/internal/repositories"
)

type ECRRepository interface {
	PushImage(ctx context.Context, imageName string) (string, error)
	GetAuthToken(ctx context.Context) (*repositories.ECRAuth, error)
}

type DockerService interface {
	BuildImage(dockerfilePath, imageName string, ecrAuth *repositories.ECRAuth) error
}

type ECRService interface {
    BuildAndPushToECR(ctx context.Context, repoFullName, accessToken string) (string, error)
}

type EKSService interface {
	DeployToEKS(ctx context.Context, imageName, appName string, userId string, containerListensOnPort int32, replicas *int32,
		cpu *string,
		memory *string) error
}

// type EKSInfoService interface {
// 	(ctx context.Context) (string, error)
// }