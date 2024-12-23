// services/monitoring_service.go
package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type MetricData struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type MetricResponse struct {
	CPUUtilization    []MetricData `json:"cpuUtilization"`
	MemoryUtilization []MetricData `json:"memoryUtilization"`
	NetworkIn         []MetricData `json:"networkIn"`
	NetworkOut        []MetricData `json:"networkOut"`
}

type MonitoringService interface {
	GetMetrics(ctx context.Context, clusterName, namespace, appName string, startTime, endTime time.Time) (*MetricResponse, error)
}

type MonitoringServiceImpl struct {
	cfg aws.Config
}

func NewMonitoringService(cfg aws.Config) MonitoringService {
	return &MonitoringServiceImpl{cfg: cfg}
}

func (s *MonitoringServiceImpl) GetMetrics(ctx context.Context, clusterName, namespace, appName string, startTime, endTime time.Time) (*MetricResponse, error) {
	cwClient := cloudwatch.NewFromConfig(s.cfg)

	// Use the correct namespace
	metricNamespace := "ContainerInsights"


	// Define metric queries with updated dimensions
	queries := []struct {
		name       string
		metricName string
		stat       string
		dimensions []types.Dimension
	}{
		{
			name:       "CPUUtilization",
			metricName: "container_cpu_utilization",
			stat:       "Average",
			dimensions: []types.Dimension{
				{Name: aws.String("ClusterName"), Value: aws.String(clusterName)},
				{Name: aws.String("Namespace"), Value: aws.String(namespace)},
				{Name: aws.String("PodName"), Value: aws.String(appName)},
				{Name: aws.String("ContainerName"), Value: aws.String(appName)},
			},
		},
		{
			name:       "MemoryUtilization",
			metricName: "container_memory_utilization",
			stat:       "Average",
			dimensions: []types.Dimension{
				{Name: aws.String("ClusterName"), Value: aws.String(clusterName)},
				{Name: aws.String("Namespace"), Value: aws.String(namespace)},
				{Name: aws.String("PodName"), Value: aws.String(appName)},
				{Name: aws.String("ContainerName"), Value: aws.String(appName)},
				// Optionally include 'FullPodName' if you want to target a specific pod instance
			},
		},
		// Repeat for other metrics (MemoryUtilization, NetworkIn, NetworkOut)
	}

	// Build metric data queries
	var metricDataQueries []types.MetricDataQuery
	for i, q := range queries {
		metricDataQueries = append(metricDataQueries, types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String(metricNamespace),
					MetricName: aws.String(q.metricName),
					Dimensions: q.dimensions,
				},
				Period: aws.Int32(300),
					Stat:   aws.String(q.stat),
			},
		})
	}

	// Log the constructed queries for debugging
	for _, q := range metricDataQueries {
		fmt.Printf("\nMetric Query:\nID: %s\nNamespace: %s\nMetricName: %s\nDimensions:\n",
			aws.ToString(q.Id), aws.ToString(q.MetricStat.Metric.Namespace), aws.ToString(q.MetricStat.Metric.MetricName))
		for _, dim := range q.MetricStat.Metric.Dimensions {
			fmt.Printf("  %s: %s\n", aws.ToString(dim.Name), aws.ToString(dim.Value))
		}
	}

	// Get metric data
	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
			MetricDataQueries: metricDataQueries,
	}

	output, err := cwClient.GetMetricData(ctx, input)
	if err != nil {
		fmt.Printf("Error fetching metric data: %v\n", err)
		return nil, fmt.Errorf("failed to get metric data: %w", err)
	}

	// Process results
	result := &MetricResponse{}
	for i, metricResult := range output.MetricDataResults {
		var metrics []MetricData
		// Create metrics slice
		for j, timestamp := range metricResult.Timestamps {
			metrics = append(metrics, MetricData{
				Timestamp: timestamp,
				Value:     metricResult.Values[j],
			})
		}

		// Sort metrics by timestamp
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Timestamp.Before(metrics[j].Timestamp)
		})

		switch queries[i].name {
		case "CPUUtilization":
			result.CPUUtilization = metrics
		case "MemoryUtilization":
			result.MemoryUtilization = metrics
		case "NetworkIn":
			result.NetworkIn = metrics
		case "NetworkOut":
			result.NetworkOut = metrics
		}
	}

	return result, nil
}