// services/eks_service.go
package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

type EKSServiceImpl struct {
	cfg aws.Config
}

func NewEKSService(ctx context.Context) (EKSService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &EKSServiceImpl{cfg: cfg}, nil
}

// Implement the DeployToEKS method for EKSServiceImpl
func (s *EKSServiceImpl) DeployToEKS(ctx context.Context, imageName, appName string, containerListensOnPort int32) error {
	clusterName := "paas-1"

	if err := s.ensureNodeGroupHasNodes(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to ensure node group has nodes: %w", err)
	}

	clientset, err := s.getKubernetesClientset(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes clientset: %w", err)
	}

	deployment := s.createDeployment(appName, imageName, containerListensOnPort)
	if _, err := clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	service := s.createService(appName, containerListensOnPort)
	if _, err := clientset.CoreV1().Services("default").Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

func (s *EKSServiceImpl) ensureNodeGroupHasNodes(ctx context.Context, clusterName string) error {
	eksClient := eks.NewFromConfig(s.cfg)
	asgClient := autoscaling.NewFromConfig(s.cfg)

	nodeGroups, err := eksClient.ListNodegroups(ctx, &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	})
	if err != nil {
		return fmt.Errorf("failed to list node groups: %w", err)
	}

	if len(nodeGroups.Nodegroups) == 0 {
		return fmt.Errorf("no node groups found for cluster %s", clusterName)
	}

	nodeGroupName := nodeGroups.Nodegroups[0]

	nodeGroup, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	})
	if err != nil {
		return fmt.Errorf("failed to describe node group: %w", err)
	}

	if nodeGroup.Nodegroup.ScalingConfig.DesiredSize != nil && *nodeGroup.Nodegroup.ScalingConfig.DesiredSize == 0 {
		asgName := *nodeGroup.Nodegroup.Resources.AutoScalingGroups[0].Name

		describeASGOutput, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []string{asgName},
		})
		if err != nil {
			return fmt.Errorf("failed to describe Auto Scaling Group: %w", err)
		}

		if len(describeASGOutput.AutoScalingGroups) == 0 {
			return fmt.Errorf("Auto Scaling Group %s not found", asgName)
		}

		asg := describeASGOutput.AutoScalingGroups[0]

		if *asg.MinSize == 0 || *asg.MaxSize == 0 {
			_, err = asgClient.UpdateAutoScalingGroup(ctx, &autoscaling.UpdateAutoScalingGroupInput{
				AutoScalingGroupName: &asgName,
				MinSize:              aws.Int32(1),
				MaxSize:              aws.Int32(1),
			})
			if err != nil {
				return fmt.Errorf("failed to update Auto Scaling Group: %w", err)
			}
		}

		_, err = asgClient.SetDesiredCapacity(ctx, &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: &asgName,
			DesiredCapacity:      aws.Int32(1),
		})
		if err != nil {
			return fmt.Errorf("failed to set desired capacity: %w", err)
		}

		return s.waitForNodes(ctx, clusterName)
	}

	return nil
}

func (s *EKSServiceImpl) waitForNodes(ctx context.Context, clusterName string) error {
	clientset, err := s.getKubernetesClientset(ctx, clusterName)
	if err != nil {
		return err
	}

	for i := 0; i < 30; i++ {
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		if len(nodes.Items) > 0 {
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
						return nil
					}
				}
			}
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for nodes to be ready")
}

func (s *EKSServiceImpl) getKubernetesClientset(ctx context.Context, clusterName string) (*kubernetes.Clientset, error) {
	eksClient := eks.NewFromConfig(s.cfg)

	describeClusterOutput, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: &clusterName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe EKS cluster: %w", err)
	}

	clusterCA, err := base64.StdEncoding.DecodeString(*describeClusterOutput.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cluster CA: %w", err)
	}

	generator, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create token generator: %w", err)
	}

	token, err := generator.GetWithOptions(&token.GetTokenOptions{
		ClusterID: clusterName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	restConfig := &rest.Config{
		Host:        *describeClusterOutput.Cluster.Endpoint,
		BearerToken: token.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: clusterCA,
		},
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

func (s *EKSServiceImpl) createDeployment(appName, imageName string, containerListensOnPort int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: aws.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  appName,
							Image: imageName,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: containerListensOnPort,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *EKSServiceImpl) createService(appName string, containerListensOnPort int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": appName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(int(containerListensOnPort)),
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
}
