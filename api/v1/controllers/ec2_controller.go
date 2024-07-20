package controllers

import (
	"encoding/json"
	"net/http"
	"paas-backend/internal/services"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type EC2CreateRequest struct {
	InstanceType string `json:"instance_type"`
	ImageID      string `json:"image_id"`
}

func CreateEC2InstanceHandler(w http.ResponseWriter, r *http.Request) {
	var req EC2CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(req.ImageID),
		InstanceType: types.InstanceType(req.InstanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	ctx := r.Context()
	result, err := services.EC2Client.RunInstances(ctx, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result.Instances[0].InstanceId)
}

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	var req services.DeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := services.Deploy(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}