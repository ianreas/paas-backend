package services

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type AWSCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
}

type DeploymentRequest struct {
	AWSCredentials AWSCredentials `json:"awsCredentials"`
	DockerImage    string         `json:"dockerImage"`
	ContainerPort  int32          `json:"containerPort"`
}

var (
	ECSClient *ecs.Client
	EC2Client *ec2.Client
	RDSClient *rds.Client
)

func InitAWSServices(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	EC2Client = ec2.NewFromConfig(cfg)
	RDSClient = rds.NewFromConfig(cfg)
	ECSClient = ecs.NewFromConfig(cfg)

	return nil
}

func Deploy(ctx context.Context, req DeploymentRequest) (map[string]string, error) {
	cfg, err := createAWSConfig(ctx, req.AWSCredentials)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	instanceID, err := createEC2Instance(ctx, ec2Client)
	if err != nil {
		return nil, err
	}

	ecsClient := ecs.NewFromConfig(cfg)
	clusterName := "my-cluster"
	if err := createECSCluster(ctx, ecsClient, clusterName); err != nil {
		return nil, err
	}

	taskDefArn, err := createECSTaskDefinition(ctx, ecsClient, req.DockerImage, req.ContainerPort)
	if err != nil {
		return nil, err
	}

	if _, err := createECSService(ctx, ecsClient, clusterName, taskDefArn); err != nil {
		return nil, err
	}

	return map[string]string{
		"message":    "Deployment successful",
		"instanceId": instanceID,
	}, nil
}

func createAWSConfig(ctx context.Context, creds AWSCredentials) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(creds.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			"",
		)),
	)
}

func createEC2Instance(ctx context.Context, client *ec2.Client) (string, error) {
	result, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String("ami-0c55b159cbfafe1f0"), // Amazon Linux 2 AMI ID
		InstanceType: types.InstanceTypeT2Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil {
		return "", err
	}
	return *result.Instances[0].InstanceId, nil
}

func createECSCluster(ctx context.Context, client *ecs.Client, clusterName string) error {
	_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	return err
}

func createECSTaskDefinition(ctx context.Context, client *ecs.Client, dockerImage string, containerPort int32) (string, error) {
	result, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("my-task-family"),
		ContainerDefinitions: []ecsTypes.ContainerDefinition{
			{
				Name:  aws.String("my-container"),
				Image: aws.String(dockerImage),
				PortMappings: []ecsTypes.PortMapping{
					{
						ContainerPort: aws.Int32(containerPort),
						HostPort:      aws.Int32(80),
					},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return *result.TaskDefinition.TaskDefinitionArn, nil
}

func createECSService(ctx context.Context, client *ecs.Client, clusterName, taskDefinitionArn string) (string, error) {
	result, err := client.CreateService(ctx, &ecs.CreateServiceInput{
		Cluster:        aws.String(clusterName),
		ServiceName:    aws.String("my-service"),
		TaskDefinition: aws.String(taskDefinitionArn),
		DesiredCount:   aws.Int32(1),
		LaunchType:     ecsTypes.LaunchTypeEc2,
	})
	if err != nil {
		return "", err
	}
	return *result.Service.ServiceArn, nil
}
