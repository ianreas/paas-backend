package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func main() {
	// Load the AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"AKIAYS2NQGBQSSNQNFN7",
			"00SmMjSVGgNqeIlpPmQ8Qno8suxNvNae7EEmzIcV",
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	// Create an ECS client
	client := ecs.NewFromConfig(cfg)

	// Create an ECS cluster
	clusterName := "my-go-cluster"
	_, err = client.CreateCluster(context.TODO(), &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		log.Fatalf("Failed to create cluster, %v", err)
	}
	fmt.Printf("Created cluster: %s\n", clusterName)

	// Register a task definition
	taskDefFamily := "my-go-task"
	taskDefResponse, err := client.RegisterTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(taskDefFamily),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:  aws.String("my-container"),
				Image: aws.String("muhammedik11/paas-frontend:meow"), // Replace with your image
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(80),
						HostPort:      aws.Int32(80),
					},
				},
				Memory: aws.Int32(512), // Specify the memory limit in MiB
			},
		},
		RequiresCompatibilities: []types.Compatibility{types.CompatibilityEc2},
		NetworkMode:             types.NetworkModeAwsvpc,
		Cpu:                     aws.String("256"),
		Memory:                  aws.String("512"),
	})
	if err != nil {
		log.Fatalf("Failed to register task definition, %v", err)
	}
	fmt.Printf("Registered task definition: %s\n", *taskDefResponse.TaskDefinition.TaskDefinitionArn)

	// Run a task
	runTaskResponse, err := client.RunTask(context.TODO(), &ecs.RunTaskInput{
		Cluster:        aws.String(clusterName),
		TaskDefinition: taskDefResponse.TaskDefinition.TaskDefinitionArn,
		LaunchType:     types.LaunchTypeEc2,
		Count:          aws.Int32(1),
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				AssignPublicIp: types.AssignPublicIpEnabled,
				Subnets:        []string{"subnet-083c2fe15284cec17"}, // Replace with your subnet ID
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to run task, %v", err)
	}
	fmt.Printf("Started task: %s\n", *runTaskResponse.Tasks[0].TaskArn)
}