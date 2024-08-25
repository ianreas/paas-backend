// services/eks_service.go
package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"

	"log"
)

type EKSServiceImpl struct {
	cfg aws.Config
}

func NewEKSService(ctx context.Context) (EKSService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	//cfg.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody

	return &EKSServiceImpl{cfg: cfg}, nil
}

// Implement the DeployToEKS method for EKSServiceImpl
func (s *EKSServiceImpl) DeployToEKS(ctx context.Context, imageName, appName string, containerListensOnPort int32) error {
	clusterName := "paas-1"

	log.Printf("Starting deployment to EKS for app: %s", appName)

	if err := s.checkServiceQuotas(ctx); err != nil {
        return fmt.Errorf("failed to check service quotas: %w", err)
    }

	if err := s.ensureNodeGroupHasNodes(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to ensure node group has nodes: %w", err)
	}

	clientset, err := s.getKubernetesClientset(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes clientset: %w", err)
	}

	if err := s.createOrUpdateDeployment(ctx, clientset, appName, imageName, containerListensOnPort); err != nil {
		return err
	}

	if err := s.createOrUpdateService(ctx, clientset, appName, containerListensOnPort); err != nil {
		return err
	}

	return nil
}


func (s *EKSServiceImpl) waitForNodes(ctx context.Context, clusterName string) error {
	clientset, err := s.getKubernetesClientset(ctx, clusterName)
	if err != nil {
		return err
	}

	for i := 0; i < 60; i++ { // Increase the number of retries
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		if len(nodes.Items) > 0 {
			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
						log.Printf("Node %s is ready", node.Name)
						return nil
					}
				}
			}
		}

		log.Printf("No ready nodes found yet, waiting...")
		time.Sleep(20 * time.Second) // Increased sleep time for stability
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

// creates a kubernetes deployment in eks
// takes the appName, imageName, and the port as arguments
// if the deployment with that appName exists in the default namespace, just updates the deployment
func (s *EKSServiceImpl) createOrUpdateDeployment(ctx context.Context, clientset *kubernetes.Clientset, appName, imageName string, containerListensOnPort int32) error {
	deployment := &appsv1.Deployment{
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

	existingDeployment, err := clientset.AppsV1().Deployments("default").Get(ctx, appName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create deployment: %w", err)
			}
			fmt.Printf("Created new deployment: %s\n", appName)
		} else {
			return fmt.Errorf("failed to check existing deployment: %w", err)
		}
	} else {
		existingDeployment.Spec = deployment.Spec
		_, err = clientset.AppsV1().Deployments("default").Update(ctx, existingDeployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
		fmt.Printf("Updated existing deployment: %s\n", appName)
	}

	return nil
}

func (s *EKSServiceImpl) createOrUpdateService(ctx context.Context, clientset *kubernetes.Clientset, appName string, containerListensOnPort int32) error {
	service := &corev1.Service{
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

	existingService, err := clientset.CoreV1().Services("default").Get(ctx, appName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			_, err = clientset.CoreV1().Services("default").Create(ctx, service, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}
			fmt.Printf("Created new service: %s\n", appName)
		} else {
			return fmt.Errorf("failed to check existing service: %w", err)
		}
	} else {
		existingService.Spec = service.Spec
		_, err = clientset.CoreV1().Services("default").Update(ctx, existingService, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}
		fmt.Printf("Updated existing service: %s\n", appName)
	}

	return nil
}

func (s *EKSServiceImpl) ensureNodeGroupHasNodes(ctx context.Context, clusterName string) error {
	eksClient := eks.NewFromConfig(s.cfg)
	asgClient := autoscaling.NewFromConfig(s.cfg)
	ec2Client := ec2.NewFromConfig(s.cfg)

	log.Printf("Ensuring node group has nodes for cluster: %s", clusterName)

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
	log.Printf("Found node group: %s", nodeGroupName)

	nodeGroup, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	})
	if err != nil {
		return fmt.Errorf("failed to describe node group: %w", err)
	}

	log.Printf("Node group %s details: Desired Size: %d, Min Size: %d, Max Size: %d",
	nodeGroupName,
	*nodeGroup.Nodegroup.ScalingConfig.DesiredSize,
	*nodeGroup.Nodegroup.ScalingConfig.MinSize,
	*nodeGroup.Nodegroup.ScalingConfig.MaxSize)


	asgName := *nodeGroup.Nodegroup.Resources.AutoScalingGroups[0].Name

	if *nodeGroup.Nodegroup.ScalingConfig.DesiredSize == 0 {
		log.Printf("Node group has no desired nodes, scaling up")

		// Update the Auto Scaling Group
		_, err = asgClient.UpdateAutoScalingGroup(ctx, &autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: &asgName,
			DesiredCapacity:      aws.Int32(1),
			MinSize:              aws.Int32(1),
			MaxSize:              aws.Int32(1),
		})

		err = s.checkEC2Errors(ctx)
		if err != nil {
			return fmt.Errorf("failed to check ec2 errors: %w", err)
		}

		if err != nil {
			return fmt.Errorf("failed to update Auto Scaling Group: %w", err)
		}

		log.Printf("Auto Scaling Group updated, waiting for instance to be launched")

		// Wait for the instance to be launched
		err = s.waitForASGInstance(ctx, asgClient, ec2Client, asgName)
		if err != nil {
			return fmt.Errorf("failed waiting for ASG instance: %w", err)
		}

		log.Printf("Instance launched successfully")
	}

	// Wait for the instance to be launched
	err = s.waitForASGInstance(ctx, asgClient, ec2Client, asgName)
	if err != nil {
		return fmt.Errorf("failed waiting for ASG instance: %w", err)
	}

	

	log.Printf("Waiting for nodes to be ready")
	return s.waitForNodes(ctx, clusterName) // Changed this line
}

func (s *EKSServiceImpl) waitForASGInstance(ctx context.Context, asgClient *autoscaling.Client, ec2Client *ec2.Client, asgName string) error {
	for i := 0; i < 30; i++ { // Wait for up to 15 minutes
		asgOutput, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []string{asgName},
		})
		log.Printf("ASG %s has %d instances", asgName, len(asgOutput.AutoScalingGroups[0].Instances))
		if err != nil {
			return fmt.Errorf("failed to describe Auto Scaling Group: %w", err)
		}

		if len(asgOutput.AutoScalingGroups) == 0 || len(asgOutput.AutoScalingGroups[0].Instances) == 0 {
			log.Printf("No instances in ASG yet, waiting...")
			time.Sleep(30 * time.Second)
			continue
		}

		instanceId := *asgOutput.AutoScalingGroups[0].Instances[0].InstanceId
		instanceOutput, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceId},
		})
		if err != nil {
			return fmt.Errorf("failed to describe EC2 instance: %w", err)
		}

		if len(instanceOutput.Reservations) > 0 && len(instanceOutput.Reservations[0].Instances) > 0 {
			instance := instanceOutput.Reservations[0].Instances[0]
			if instance.State.Name == "running" {
				log.Printf("Instance %s is running", instanceId)
				return nil
			}
		}

		log.Printf("Instance not yet running, waiting...")
		time.Sleep(30 * time.Second)
	}

	return fmt.Errorf("timeout waiting for ASG instance to be running")
}


func (s *EKSServiceImpl) checkServiceQuotas(ctx context.Context) error {
    sqClient := servicequotas.NewFromConfig(s.cfg)

    quotas := []struct {
        serviceName string
        quotaCode   string
        description string
    }{
        {"ec2", "L-1216C47A", "Running On-Demand Standard (A, C, D, H, I, M, R, T, Z) instances"},
        {"autoscaling", "L-CDE20ADC", "Auto Scaling groups per region"},
        {"eks", "L-1194D53C", "Clusters per Region"},
    }

    for _, q := range quotas {
        output, err := sqClient.GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
            ServiceCode: aws.String(q.serviceName),
            QuotaCode:   aws.String(q.quotaCode),
        })
        if err != nil {
            log.Printf("Error checking quota for %s - %s: %v", q.serviceName, q.description, err)
        } else {
            log.Printf("Quota for %s - %s: %f", q.serviceName, q.description, *output.Quota.Value)
        }
    }

    return nil
}


func (s *EKSServiceImpl) checkEC2Errors(ctx context.Context) error {
    ec2Client := ec2.NewFromConfig(s.cfg)

    output, err := ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
        IncludeAllInstances: aws.Bool(true),
    })
    if err != nil {
        return fmt.Errorf("failed to describe instance status: %w", err)
    }

    for _, status := range output.InstanceStatuses {
        if status.InstanceState.Name == "pending" || status.InstanceState.Name == "running" {
            continue
        }
        log.Printf("Instance %s is in state %s", *status.InstanceId, status.InstanceState.Name)
        if len(status.Events) > 0 {
            for _, event := range status.Events {
                log.Printf("Instance %s event: %s", *status.InstanceId, *event.Description)
            }
        }
    }

    return nil
}