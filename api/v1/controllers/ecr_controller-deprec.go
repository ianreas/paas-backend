package controllers

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"paas-backend/internal/services"
// 	"path/filepath"
// )

// // structs are like classes but without the methods 
// // structs in go are a little weird but you can define methods on them separately
// // and pass in data from structs into those methods
// type ECRController struct {
// 	ecrService services.ECRService
// 	eksService services.EKSService
// }

// // a factory constructor function that creates a new ECRController instance
// // it takes two parameters, ecrService and eksService and then passes them to the struct
// // and then initalizes an instance of that ECRController struct with those parameters
// // but it doesnt return the struct itself, it creates an instance and then returns a pointer to it
// // Note: *ECRController is a pointer type, its used when you want to work with a reference to the struct
// // functions that take that type as a parameter, can directly modify the original instance
// func NewECRController(ecrService services.ECRService, eksService services.EKSService) *ECRController {
// 	// & is used to return an address of the ECRController instance
// 	// it creates a pointer to an existing struct (this also creates the instance itself)
// 	return &ECRController{
// 		ecrService: ecrService,
// 		eksService: eksService,
// 	}
// }

// type BuildAndPushRequest struct {
// 	RepoFullName string `json:"repoFullName"`
// 	AccessToken  string `json:"accessToken"`
// }

// func (c *ECRController) BuildAndPushToECRApiHandler(w http.ResponseWriter, r *http.Request) {
// 	var req BuildAndPushRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	ecrImageName, err := c.ecrService.BuildAndPushToECR(r.Context(), req.RepoFullName, req.AccessToken)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error building and pushing to ECR: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// Extract the app name from the repo full name
// 	appName := filepath.Base(req.RepoFullName)

// 	// Deploy to EKS
// 	err = c.eksService.DeployToEKS(r.Context(), ecrImageName, appName, 3000)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error deploying to EKS: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{
// 		"message":      "Image built, pushed to ECR, and deployed to EKS successfully",
// 		"ecrImageName": ecrImageName,
// 		"appName":      appName,
// 	})
// }

// package controllers

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"os/exec"

// 	//"bytes"
// 	"encoding/base64"
// 	"paas-backend/internal/services"
// 	"strings"

// 	"github.com/aws/aws-sdk-go-v2/service/ec2"

// 	"github.com/aws/aws-sdk-go-v2/config"
// 	"github.com/aws/aws-sdk-go-v2/service/ecr"

// 	"github.com/aws/aws-sdk-go-v2/service/eks"

// 	"sigs.k8s.io/aws-iam-authenticator/pkg/token"

// 	appsv1 "k8s.io/api/apps/v1"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/util/intstr"
// 	"k8s.io/client-go/kubernetes"

// 	"path/filepath"

// 	"k8s.io/client-go/rest"

// 	"time"

// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/service/autoscaling"

// 	"log"

// 	k8serrors "k8s.io/apimachinery/pkg/api/errors"
// )

// // g

// type BuildAndPushRequest struct {
// 	RepoFullName string `json:"repoFullName"`
// 	AccessToken  string `json:"accessToken"`
// }

// // the actual api handler function
// func BuildAndPushToECR(w http.ResponseWriter, r *http.Request) {
// 	// this is just declaring a variable
// 	var req BuildAndPushRequest

// 	// json.NewDecoder(r.Body).Decode(&req) => this part is decoding the request body into the req variable, like
// 	// const req: BuildAndPushRequest = response.data; in typescript.
// 	// we pass a pointer &req into Decode() so that allows the Decode() function to modify the req variable directly.
// 	// The error checking (if err := ... ; err != nil):
// 	// This is similar to a try-catch block in TypeScript. It's checking if there was an error during the JSON parsing.
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		// this is like res.status(400).send('Invalid request body'); (sending error response)
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)

// 		// if there was an error, we return from the function early
// 		return
// 	}

// 	// Clone repository
// 	// we do this by creating a command and then running using exec.Run()
// 	// its also wrapped into the try catch block like the rest of the code using if err := ... ; err != nil
// 	repoDir := filepath.Join(os.TempDir(), strings.ReplaceAll(req.RepoFullName, "/", "_"))

// 	// Ensure the directory is removed before cloning
// 	if err := os.RemoveAll(repoDir); err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to remove existing directory: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	cloneCmd := exec.Command("git", "clone", fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", req.AccessToken, req.RepoFullName), repoDir)
// 	// if err := cloneCmd.Run(); err != nil {
// 	// 	http.Error(w, fmt.Sprintf("Failed to clone repository: %v", err), http.StatusInternalServerError)
// 	// 	return
// 	// }
// 	// defer os.RemoveAll(repoDir) // Clean up after we're done
// 	var out, errOut strings.Builder
// 	cloneCmd.Stdout = &out
// 	cloneCmd.Stderr = &errOut

// 	if err := cloneCmd.Run(); err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to clone repository: %v. Output: %s. Error: %s", err, out.String(), errOut.String()), http.StatusInternalServerError)
// 		return
// 	}
// 	defer os.RemoveAll(repoDir) // Clean up after we're done

// 	// Find Dockerfile
// 	dockerfilePath, err := services.FindDockerfile(repoDir)
// 	if err != nil {
// 		http.Error(w, "Dockerfile not found", http.StatusBadRequest)
// 		return
// 	}

// 	// Build Docker image
// 	imageName := filepath.Base(req.RepoFullName)
// 	buildCmd := exec.Command("docker", "build", "-f", dockerfilePath, "-t", fmt.Sprintf("%s:latest", imageName), filepath.Dir(dockerfilePath))
// 	if err := buildCmd.Run(); err != nil {
// 		http.Error(w, "Failed to build Docker image", http.StatusInternalServerError)
// 		return
// 	}

// 	//Push to ECR
// 	ecrImageName, err := pushToECR(r.Context(), imageName)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to push to ECR: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	if err := deployToEKS(r.Context(), ecrImageName, imageName, 3000); err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to deploy to EKS: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{"message": "Image built and pushed successfully"})
// }

// func pushToECR(ctx context.Context, imageName string) (string, error) {
// 	ecrImageName := fmt.Sprintf("590183673953.dkr.ecr.us-east-1.amazonaws.com/my-express-app:%s", imageName)

// 	// Load AWS configuration
// 	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to load AWS config: %w", err)
// 	}

// 	// Create ECR client
// 	client := ecr.NewFromConfig(cfg)

// 	// Get ECR authorization token
// 	authOutput, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
// 	if err != nil {
// 		return "", fmt.Errorf("failed to get ECR auth token: %w", err)
// 	}

// 	// Decode auth token and extract username/password
// 	authToken, err := base64.StdEncoding.DecodeString(*authOutput.AuthorizationData[0].AuthorizationToken)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to decode auth token: %w", err)
// 	}
// 	parts := strings.SplitN(string(authToken), ":", 2)
// 	if len(parts) != 2 {
// 		return "", fmt.Errorf("invalid auth token format")
// 	}
// 	username, password := parts[0], parts[1]

// 	// Login to ECR
// 	loginCmd := exec.Command("docker", "login", "--username", username, "--password-stdin", "590183673953.dkr.ecr.us-east-1.amazonaws.com")
// 	loginCmd.Stdin = strings.NewReader(password)
// 	loginOut, err := loginCmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to login to ECR: %w, output: %s", err, loginOut)
// 	}

// 	// Tag the image
// 	tagCmd := exec.Command("docker", "tag", fmt.Sprintf("%s:latest", imageName), ecrImageName)
// 	tagOut, err := tagCmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to tag image: %w, output: %s", err, tagOut)
// 	}

// 	// Push the image
// 	pushCmd := exec.Command("docker", "push", ecrImageName)
// 	pushOut, err := pushCmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to push image to ECR: %w, output: %s", err, pushOut)
// 	}

// 	return ecrImageName, nil
// }

// func int32Ptr(i int32) *int32 { return &i }

// func deployToEKS(ctx context.Context, imageName, appName string, containerListensOnPort int32) error {
// 	// Load the AWS configuration
// 	cfg, err := config.LoadDefaultConfig(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to load AWS config: %w", err)
// 	}

// 	// Create EKS client
// 	eksClient := eks.NewFromConfig(cfg)

// 	clusterName := "paas-1"
// 	describeClusterOutput, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
// 		Name: &clusterName,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to describe EKS cluster: %w", err)
// 	}

// 	if err := ensureNodeGroupHasNodes(ctx, cfg, clusterName); err != nil {
// 		return fmt.Errorf("failed to ensure node group has nodes: %w", err)
// 	}

// 	clusterCA, err := base64.StdEncoding.DecodeString(*describeClusterOutput.Cluster.CertificateAuthority.Data)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode cluster CA: %w", err)
// 	}

// 	generator, err := token.NewGenerator(true, false)
// 	if err != nil {
// 		return fmt.Errorf("failed to create token generator: %w", err)
// 	}

// 	token, err := generator.GetWithOptions(&token.GetTokenOptions{
// 		ClusterID: clusterName,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to get token: %w", err)
// 	}

// 	// Create the REST config for the Kubernetes clientset
// 	restConfig := &rest.Config{
// 		Host:        *describeClusterOutput.Cluster.Endpoint,
// 		BearerToken: token.Token,
// 		TLSClientConfig: rest.TLSClientConfig{
// 			CAData: clusterCA,
// 		},
// 	}

// 	// Create the clientset
// 	clientset, err := kubernetes.NewForConfig(restConfig)
// 	if err != nil {
// 		return fmt.Errorf("failed to create clientset: %w", err)
// 	}

// 	// Define the deployment
// 	deployment := &appsv1.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: appName,
// 		},
// 		Spec: appsv1.DeploymentSpec{
// 			Replicas: int32Ptr(1),
// 			Selector: &metav1.LabelSelector{
// 				MatchLabels: map[string]string{
// 					"app": appName,
// 				},
// 			},
// 			Template: corev1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: map[string]string{
// 						"app": appName,
// 					},
// 				},
// 				Spec: corev1.PodSpec{
// 					Containers: []corev1.Container{
// 						{
// 							Name:  appName,
// 							Image: imageName,
// 							Ports: []corev1.ContainerPort{
// 								{
// 									ContainerPort: containerListensOnPort,
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	// Create the deployment
// 	// Try to get the existing deployment
// 	existingDeployment, err := clientset.AppsV1().Deployments("default").Get(ctx, appName, metav1.GetOptions{})
// 	if err != nil {
// 		if k8serrors.IsNotFound(err) {
// 			// Deployment doesn't exist, create a new one
// 			_, err = clientset.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
// 			if err != nil {
// 				return fmt.Errorf("failed to create deployment: %w", err)
// 			}
// 			fmt.Printf("Created new deployment: %s\n", appName)
// 		} else {
// 			return fmt.Errorf("failed to check existing deployment: %w", err)
// 		}
// 	} else {
// 		// Deployment exists, update it
// 		existingDeployment.Spec = deployment.Spec
// 		_, err = clientset.AppsV1().Deployments("default").Update(ctx, existingDeployment, metav1.UpdateOptions{})
// 		if err != nil {
// 			return fmt.Errorf("failed to update deployment: %w", err)
// 		}
// 		fmt.Printf("Updated existing deployment: %s\n", appName)
// 	}

// 	// Define the service
// 	service := &corev1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: appName,
// 		},
// 		Spec: corev1.ServiceSpec{
// 			Selector: map[string]string{
// 				"app": appName,
// 			},
// 			Ports: []corev1.ServicePort{
// 				{
// 					Port:       80,
// 					TargetPort: intstr.FromInt(int(containerListensOnPort)),
// 				},
// 			},
// 			Type: corev1.ServiceTypeLoadBalancer,
// 		},
// 	}

// 	// Create the service
// 	existingService, err := clientset.CoreV1().Services("default").Get(ctx, appName, metav1.GetOptions{})
// 	if err != nil {
// 		if k8serrors.IsNotFound(err) {
// 			// Service doesn't exist, create a new one
// 			_, err = clientset.CoreV1().Services("default").Create(ctx, service, metav1.CreateOptions{})
// 			if err != nil {
// 				return fmt.Errorf("failed to create service: %w", err)
// 			}
// 			fmt.Printf("Created new service: %s\n", appName)
// 		} else {
// 			return fmt.Errorf("failed to check existing service: %w", err)
// 		}
// 	} else {
// 		// Service exists, update it
// 		existingService.Spec = service.Spec
// 		_, err = clientset.CoreV1().Services("default").Update(ctx, existingService, metav1.UpdateOptions{})
// 		if err != nil {
// 			return fmt.Errorf("failed to update service: %w", err)
// 		}
// 		fmt.Printf("Updated existing service: %s\n", appName)
// 	}

// 	return nil
// }

// // 1. decode the request body into a struct that stores the repoFullName and accessToken.
// // 		- maybe i should also ask the users to provide the memory and cpu limits for the container in the same api request
// //		- so i can build the image, push to ecr and deploy to eks all from 1 request
// // 2. clone the repository using the accessToken
// // 3. find the Dockerfile in the cloned repository
// // 4. build the Docker image using the Dockerfile
// // 5. push the Docker image to ECR
// // I AM HERE
// // 6.

// func waitForNodes(ctx context.Context, cfg aws.Config, clusterName string) error {
// 	clientset, err := getKubernetesClientset(ctx, cfg, clusterName)
// 	if err != nil {
// 		return fmt.Errorf("failed to get Kubernetes clientset: %w", err)
// 	}

// 	timeout := 15 * time.Minute
// 	interval := 30 * time.Second
// 	startTime := time.Now()

// 	log.Printf("Starting to wait for nodes to be ready. Timeout set to %v", timeout)

// 	for {
// 		if time.Since(startTime) > timeout {
// 			return fmt.Errorf("timeout waiting for nodes to be ready after %v", timeout)
// 		}

// 		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
// 		if err != nil {
// 			log.Printf("Error listing nodes: %v", err)
// 			time.Sleep(interval)
// 			continue
// 		}

// 		log.Printf("Found %d nodes", len(nodes.Items))

// 		readyNodes := 0
// 		for _, node := range nodes.Items {
// 			log.Printf("Node %s status:", node.Name)
// 			for _, condition := range node.Status.Conditions {
// 				if condition.Type == corev1.NodeReady {
// 					log.Printf("  Ready: %v, Reason: %s, Message: %s", condition.Status == corev1.ConditionTrue, condition.Reason, condition.Message)
// 					if condition.Status == corev1.ConditionTrue {
// 						readyNodes++
// 					}
// 				}
// 			}
// 		}

// 		log.Printf("%d/%d nodes are ready", readyNodes, len(nodes.Items))

// 		if readyNodes > 0 {
// 			log.Printf("At least one node is ready after %v", time.Since(startTime))
// 			return nil
// 		}

// 		log.Printf("No nodes are ready yet. Waiting %v before checking again...", interval)
// 		time.Sleep(interval)
// 	}
// }

// func ensureNodeGroupHasNodes(ctx context.Context, cfg aws.Config, clusterName string) error {
// 	eksClient := eks.NewFromConfig(cfg)
// 	asgClient := autoscaling.NewFromConfig(cfg)
// 	ec2Client := ec2.NewFromConfig(cfg)

// 	log.Printf("Ensuring node group has nodes for cluster: %s", clusterName)

// 	nodeGroups, err := eksClient.ListNodegroups(ctx, &eks.ListNodegroupsInput{
// 		ClusterName: &clusterName,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to list node groups: %w", err)
// 	}

// 	if len(nodeGroups.Nodegroups) == 0 {
// 		return fmt.Errorf("no node groups found for cluster %s", clusterName)
// 	}

// 	nodeGroupName := nodeGroups.Nodegroups[0]
// 	log.Printf("Found node group: %s", nodeGroupName)

// 	nodeGroup, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
// 		ClusterName:   &clusterName,
// 		NodegroupName: &nodeGroupName,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to describe node group: %w", err)
// 	}

// 	log.Printf("Node group details: Desired Size: %d, Min Size: %d, Max Size: %d",
// 		*nodeGroup.Nodegroup.ScalingConfig.DesiredSize,
// 		*nodeGroup.Nodegroup.ScalingConfig.MinSize,
// 		*nodeGroup.Nodegroup.ScalingConfig.MaxSize)

// 	if *nodeGroup.Nodegroup.ScalingConfig.DesiredSize == 0 {
// 		log.Printf("Node group has no desired nodes, scaling up")
// 		asgName := *nodeGroup.Nodegroup.Resources.AutoScalingGroups[0].Name

// 		// Update the Auto Scaling Group
// 		_, err = asgClient.UpdateAutoScalingGroup(ctx, &autoscaling.UpdateAutoScalingGroupInput{
// 			AutoScalingGroupName: &asgName,
// 			DesiredCapacity:      aws.Int32(1),
// 			MinSize:              aws.Int32(1),
// 			MaxSize:              aws.Int32(1),
// 		})
// 		if err != nil {
// 			return fmt.Errorf("failed to update Auto Scaling Group: %w", err)
// 		}

// 		log.Printf("Auto Scaling Group updated, waiting for instance to be launched")

// 		// Wait for the instance to be launched
// 		err = waitForASGInstance(ctx, asgClient, ec2Client, asgName)
// 		if err != nil {
// 			return fmt.Errorf("failed waiting for ASG instance: %w", err)
// 		}

// 		log.Printf("Instance launched successfully")
// 	}

// 	log.Printf("Waiting for nodes to be ready")
// 	return waitForNodes(ctx, cfg, clusterName)
// }

// func getKubernetesClientset(ctx context.Context, cfg aws.Config, clusterName string) (*kubernetes.Clientset, error) {
// 	// Create EKS client
// 	eksClient := eks.NewFromConfig(cfg)

// 	// Describe the cluster to get its details
// 	describeClusterOutput, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
// 		Name: &clusterName,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to describe EKS cluster: %w", err)
// 	}

// 	// Decode the cluster CA
// 	clusterCA, err := base64.StdEncoding.DecodeString(*describeClusterOutput.Cluster.CertificateAuthority.Data)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode cluster CA: %w", err)
// 	}

// 	// Create token generator
// 	generator, err := token.NewGenerator(true, false)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create token generator: %w", err)
// 	}

// 	// Generate token
// 	token, err := generator.GetWithOptions(&token.GetTokenOptions{
// 		ClusterID: clusterName,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get token: %w", err)
// 	}

// 	// Create the REST config for the Kubernetes clientset
// 	restConfig := &rest.Config{
// 		Host:        *describeClusterOutput.Cluster.Endpoint,
// 		BearerToken: token.Token,
// 		TLSClientConfig: rest.TLSClientConfig{
// 			CAData: clusterCA,
// 		},
// 	}

// 	// Create the clientset
// 	clientset, err := kubernetes.NewForConfig(restConfig)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create clientset: %w", err)
// 	}

// 	return clientset, nil
// }

// func waitForASGInstance(ctx context.Context, asgClient *autoscaling.Client, ec2Client *ec2.Client, asgName string) error {
// 	for i := 0; i < 30; i++ { // Wait for up to 15 minutes
// 		asgOutput, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
// 			AutoScalingGroupNames: []string{asgName},
// 		})
// 		if err != nil {
// 			return fmt.Errorf("failed to describe Auto Scaling Group: %w", err)
// 		}

// 		if len(asgOutput.AutoScalingGroups) == 0 || len(asgOutput.AutoScalingGroups[0].Instances) == 0 {
// 			log.Printf("No instances in ASG yet, waiting...")
// 			time.Sleep(30 * time.Second)
// 			continue
// 		}

// 		instanceId := *asgOutput.AutoScalingGroups[0].Instances[0].InstanceId
// 		instanceOutput, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
// 			InstanceIds: []string{instanceId},
// 		})
// 		if err != nil {
// 			return fmt.Errorf("failed to describe EC2 instance: %w", err)
// 		}

// 		if len(instanceOutput.Reservations) > 0 && len(instanceOutput.Reservations[0].Instances) > 0 {
// 			instance := instanceOutput.Reservations[0].Instances[0]
// 			if instance.State.Name == "running" {
// 				log.Printf("Instance %s is running", instanceId)
// 				return nil
// 			}
// 		}

// 		log.Printf("Instance not yet running, waiting...")
// 		time.Sleep(30 * time.Second)
// 	}

// 	return fmt.Errorf("timeout waiting for ASG instance to be running")
// }
