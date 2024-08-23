package services


import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type EKSInfoService struct {
	cfg aws.Config
}

func NewEKSInfoService(ctx context.Context) (*EKSInfoService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: in NewEKSInfoService: %w", err)
	}
	return &EKSInfoService{cfg: cfg}, nil
}