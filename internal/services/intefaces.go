package services

import "context"

type ECRRepository interface {
    PushImage(ctx context.Context, imageName string) (string, error)
}

type DockerService interface {
    BuildImage(dockerfilePath, imageName string) error
}

type EKSService interface {
    DeployToEKS(ctx context.Context, imageName, appName string, containerListensOnPort int32) error
}