package services

import (
	"context"
	"fmt"
	"time"
	"strings"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type LogService interface {
	StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error)
}

type LogServiceImpl struct {
	client *cloudwatchlogs.Client
}

func NewLogService(ctx context.Context) (LogService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	return &LogServiceImpl{client: client}, nil
}

func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error) {
    logChan := make(chan string)
    errChan := make(chan error, 1)

    go func() {
        defer close(logChan)
        defer close(errChan)

        // First, list the log streams for this app
        describeLogStreamsInput := &cloudwatchlogs.DescribeLogStreamsInput{
            LogGroupName:        aws.String("/aws/containerinsights/paas-1/application"),
            LogStreamNamePrefix: aws.String("ip-"),
        }

        logStreams, err := s.client.DescribeLogStreams(ctx, describeLogStreamsInput)
        if err != nil {
            errChan <- fmt.Errorf("failed to describe log streams: %w", err)
            return
        }

        if len(logStreams.LogStreams) == 0 {
            errChan <- fmt.Errorf("no log streams found for app: %s", appName)
            return
        }

        // For each log stream, get and send the logs
        for _, stream := range logStreams.LogStreams {
            // Check if the stream name contains the app name
            if !strings.Contains(*stream.LogStreamName, appName) {
                continue
            }

            params := &cloudwatchlogs.GetLogEventsInput{
                LogGroupName:  aws.String("/aws/containerinsights/paas-1/application"),
                LogStreamName: stream.LogStreamName,
                StartTime:     aws.Int64(startTime.UnixNano() / int64(time.Millisecond)),
                StartFromHead: aws.Bool(true),
            }

            paginator := cloudwatchlogs.NewGetLogEventsPaginator(s.client, params)

            for paginator.HasMorePages() {
                output, err := paginator.NextPage(ctx)
                if err != nil {
                    errChan <- err
                    return
                }

                for _, event := range output.Events {
                    select {
                    case logChan <- *event.Message:
                    case <-ctx.Done():
                        return
                    }
                }
            }
        }
    }()

    return logChan, errChan
}