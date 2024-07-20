package controllers

import (
	"encoding/json"
	"net/http"
	"paas-backend/internal/services"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/aws"
	"context"
)

type RDSCreateRequest struct {
	DBInstanceIdentifier string `json:"db_instance_identifier"`
	DBInstanceClass      string `json:"db_instance_class"`
	Engine               string `json:"engine"`
}

func CreateRDSInstanceHandler(w http.ResponseWriter, r *http.Request) {
	var req RDSCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(req.DBInstanceIdentifier),
		DBInstanceClass:      aws.String(req.DBInstanceClass),
		Engine:               aws.String(req.Engine),
		AllocatedStorage:     aws.Int32(20),
		// set this to real and put them in env file
		MasterUsername:       aws.String("admin"),
		MasterUserPassword:   aws.String("password123"),
	}

	result, err := services.RDSClient.CreateDBInstance(context.TODO(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result.DBInstance.DBInstanceIdentifier)
}