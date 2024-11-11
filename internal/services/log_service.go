// package services

// import (
// 	"context"
// 	"fmt"
// 	"strings"
// 	"time"

// 	 "github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/config"
// 	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"


//     "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

// 	 "log"
// )




// // type LogService interface {
// // 	StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error)
// // }

// type LogServiceImpl struct {
// 	client *cloudwatchlogs.Client
// }

// func NewLogService(ctx context.Context) (LogService, error) {
// 	cfg, err := config.LoadDefaultConfig(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load AWS config: %w", err)
// 	}

// 	client := cloudwatchlogs.NewFromConfig(cfg)

// 	return &LogServiceImpl{client: client}, nil
// }

// func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error) {
//     logChan := make(chan string)
//     errChan := make(chan error, 1)

//     go func() {
//         defer close(logChan)
//         defer close(errChan)

//         // First, list the log streams for this app
//         describeLogStreamsInput := &cloudwatchlogs.DescribeLogStreamsInput{
//             LogGroupName:        aws.String("/aws/containerinsights/paas-1/application"),
//             LogStreamNamePrefix: aws.String("ip-"),
//         }

//         logStreams, err := s.client.DescribeLogStreams(ctx, describeLogStreamsInput)
//         if err != nil {
//             errChan <- fmt.Errorf("failed to describe log streams: %w", err)
//             return
//         }

//         if len(logStreams.LogStreams) == 0 {
//             errChan <- fmt.Errorf("no log streams found for app: %s", appName)
//             return
//         }

//         fmt.Printf("Found %d log streams\n", len(logStreams.LogStreams))

//         streamCount := 0

//         // For each log stream, get and send the logs
//         for _, stream := range logStreams.LogStreams {
//             // Check if the stream name contains the app name
//             if !strings.Contains(*stream.LogStreamName, appName) {
//                 continue
//             }
            
//             streamCount++
//             log.Printf("Found matching stream %d: %s", streamCount, *stream.LogStreamName)

//             params := &cloudwatchlogs.GetLogEventsInput{
//                 LogGroupName:  aws.String("/aws/containerinsights/paas-1/application"),
//                 LogStreamName: stream.LogStreamName,
//                 StartTime:     aws.Int64(startTime.UnixNano() / int64(time.Millisecond)),
//                 StartFromHead: aws.Bool(true),
//             }

//             paginator := cloudwatchlogs.NewGetLogEventsPaginator(s.client, params)

//             pageCount := 0

//             for paginator.HasMorePages() {
//                 pageCount++
//                 log.Printf("Processing page %d for stream %s", pageCount, *stream.LogStreamName)


//                 output, err := paginator.NextPage(ctx)
//                 if err != nil {
//                     wrappedErr := fmt.Errorf("failed to fetch logs from stream %s (page %d): %w", 
//                         *stream.LogStreamName, 
//                         pageCount, 
//                         err,
//                     )
//                     log.Printf("Error fetching logs: %v", wrappedErr)
//                     errChan <- wrappedErr
//                     return
//                 }

//                 log.Printf("Retrieved %d events on page %d", len(output.Events), pageCount)

                

//                 for _, event := range output.Events {
//                     log.Printf("Log message: %s", *event.Message)
//                     select {
//                     case logChan <- *event.Message:
//                     case <-ctx.Done():
//                         return
//                     }
//                 }
//             }
//         }
//     }()

//     return logChan, errChan
// }


// v2. worked well. fetched past logs
// but not the new ones
// need to add functionality to list to the new logs
// func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error) {
//     logChan := make(chan string)
//     errChan := make(chan error, 1)
    
//     log.Printf("Starting to stream logs for app: %s", appName)

//     go func() {
//         defer close(logChan)
//         defer close(errChan)

//         // Try different possible log group names
//         logGroups := []string{
//             "/aws/containerinsights/paas-1/application",
//             fmt.Sprintf("/aws/eks/paas-1/cluster"),
//             fmt.Sprintf("/eks/paas-1/containers"),
//         }

//         for _, logGroupName := range logGroups {
//             log.Printf("Checking log group: %s", logGroupName)

//             // First, list the log streams for this app
//             input := &cloudwatchlogs.DescribeLogStreamsInput{
//                 LogGroupName: aws.String(logGroupName),
//                 Descending:   aws.Bool(true), // Get newest streams first
//                 OrderBy:     types.OrderByLastEventTime,
//             }

//             logStreams, err := s.client.DescribeLogStreams(ctx, input)
//             if err != nil {
//                 log.Printf("Error describing log streams for group %s: %v", logGroupName, err)
//                 continue
//             }

//             log.Printf("Found %d streams in group %s", len(logStreams.LogStreams), logGroupName)

//             // Find relevant streams
//             for _, stream := range logStreams.LogStreams {
//                 streamName := *stream.LogStreamName
//                 if !strings.Contains(strings.ToLower(streamName), strings.ToLower(appName)) {
//                     continue
//                 }

//                 log.Printf("Found matching stream: %s", streamName)

//                 // Calculate start time
//                 // Start from 1 hour ago if no specific start time is provided
//                 startTimeMs := startTime.Add(-1 * time.Hour).UnixNano() / int64(time.Millisecond)
                
//                 params := &cloudwatchlogs.GetLogEventsInput{
//                     LogGroupName:  aws.String(logGroupName),
//                     LogStreamName: aws.String(streamName),
//                     StartTime:     aws.Int64(startTimeMs),
//                     StartFromHead: aws.Bool(false), // Get newest logs first
//                     Limit:         aws.Int32(100),  // Limit results per page
//                 }

//                 log.Printf("Getting logs from stream %s starting from %v", 
//                     streamName, time.Unix(0, startTimeMs*int64(time.Millisecond)))

//                 var lastToken *string
//                 pageCount := 0
//                 emptyPages := 0

//                 for {
//                     if pageCount > 0 && lastToken == nil {
//                         break
//                     }

//                     output, err := s.client.GetLogEvents(ctx, params)
//                     if err != nil {
//                         log.Printf("Error getting log events: %v", err)
//                         break
//                     }

//                     pageCount++
//                     log.Printf("Page %d: Retrieved %d events", pageCount, len(output.Events))

//                     if len(output.Events) == 0 {
//                         emptyPages++
//                         if emptyPages >= 3 { // Break after 3 consecutive empty pages
//                             log.Printf("No more events found after %d empty pages", emptyPages)
//                             break
//                         }
//                     } else {
//                         emptyPages = 0
//                     }

//                     for _, event := range output.Events {
//                         eventTime := time.Unix(0, *event.Timestamp*int64(time.Millisecond))
//                         log.Printf("Event from %v: %s", eventTime, *event.Message)
                        
//                         select {
//                         case logChan <- *event.Message:
//                         case <-ctx.Done():
//                             return
//                         }
//                     }

//                     // Break if we've gotten back the same token
//                     if lastToken != nil && output.NextForwardToken != nil && *lastToken == *output.NextForwardToken {
//                         log.Printf("Received same token, finishing stream")
//                         break
//                     }

//                     lastToken = output.NextForwardToken
//                     params.NextToken = output.NextForwardToken

//                     // Add a small delay to avoid hitting rate limits
//                     time.Sleep(100 * time.Millisecond)
//                 }
//             }
//         }
//     }()

//     return logChan, errChan
// }


// v3.  worked well! polling for new logs 
// func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error) {
//     logChan := make(chan string)
//     errChan := make(chan error, 1)
    
//     log.Printf("Starting to stream logs for app: %s", appName)

//     go func() {
//         defer close(logChan)
//         defer close(errChan)

//         logGroups := []string{
//             "/aws/containerinsights/paas-1/application",
//             fmt.Sprintf("/aws/eks/paas-1/cluster"),
//             fmt.Sprintf("/eks/paas-1/containers"),
//         }

//         for {
//             select {
//             case <-ctx.Done():
//                 log.Printf("Context cancelled, stopping log streaming")
//                 return
//             default:
//                 for _, logGroupName := range logGroups {
//                     input := &cloudwatchlogs.DescribeLogStreamsInput{
//                         LogGroupName: aws.String(logGroupName),
//                         Descending:   aws.Bool(true),
//                         OrderBy:      types.OrderByLastEventTime,
//                     }

//                     logStreams, err := s.client.DescribeLogStreams(ctx, input)
//                     if err != nil {
//                         log.Printf("Error describing log streams for group %s: %v", logGroupName, err)
//                         continue
//                     }

//                     // Track the latest timestamp we've seen
//                     var latestTimestamp int64

//                     // Find relevant streams
//                     for _, stream := range logStreams.LogStreams {
//                         streamName := *stream.LogStreamName
//                         if !strings.Contains(strings.ToLower(streamName), strings.ToLower(appName)) {
//                             continue
//                         }

//                         log.Printf("Checking stream: %s for new logs", streamName)

//                         // Use the stored timestamp or start from the provided time
//                         startTimeMs := startTime.UnixNano() / int64(time.Millisecond)
//                         if latestTimestamp > 0 {
//                             startTimeMs = latestTimestamp
//                         }
                        
//                         params := &cloudwatchlogs.GetLogEventsInput{
//                             LogGroupName:  aws.String(logGroupName),
//                             LogStreamName: aws.String(streamName),
//                             StartTime:     aws.Int64(startTimeMs),
//                             StartFromHead: aws.Bool(false),
//                             Limit:         aws.Int32(100),
//                         }

//                         var lastToken *string
//                         newEvents := false

//                         for {
//                             if lastToken != nil && params.NextToken != nil && *lastToken == *params.NextToken {
//                                 break
//                             }

//                             output, err := s.client.GetLogEvents(ctx, params)
//                             if err != nil {
//                                 log.Printf("Error getting log events: %v", err)
//                                 break
//                             }

//                             for _, event := range output.Events {
//                                 newEvents = true
//                                 eventTime := *event.Timestamp
//                                 if eventTime > latestTimestamp {
//                                     latestTimestamp = eventTime
//                                 }
                                
//                                 eventTimeFormatted := time.Unix(0, eventTime*int64(time.Millisecond))
//                                 log.Printf("New event from %v: %s", eventTimeFormatted, *event.Message)
                                
//                                 select {
//                                 case logChan <- *event.Message:
//                                 case <-ctx.Done():
//                                     return
//                                 }
//                             }

//                             lastToken = params.NextToken
//                             params.NextToken = output.NextForwardToken
//                         }

//                         if !newEvents {
//                             log.Printf("No new events found in stream %s", streamName)
//                         }
//                     }
//                 }
//             }

//             // Wait before checking for new logs
//             select {
//             case <-ctx.Done():
//                 return
//             case <-time.After(5 * time.Second): // Poll every 5 seconds for new logs
//                 log.Printf("Checking for new logs...")
//                 continue
//             }
//         }
//     }()

//     return logChan, errChan
// }


// nice v4. 
// logs work well and the new logs show up correctly in the ui. 
// func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, startTime time.Time) (<-chan string, <-chan error) {
//     logChan := make(chan string)
//     errChan := make(chan error, 1)
    
//     log.Printf("Starting to stream logs for app: %s", appName)

//     go func() {
//         defer close(logChan)
//         defer close(errChan)

//         // Keep track of the latest timestamp we've seen
//         var latestTimestamp int64 = startTime.UnixNano() / int64(time.Millisecond)
//         seenLogs := make(map[string]bool)

//         for {
//             select {
//             case <-ctx.Done():
//                 return
//             default:
//                 input := &cloudwatchlogs.DescribeLogStreamsInput{
//                     LogGroupName: aws.String("/aws/containerinsights/paas-1/application"),
//                     Descending:   aws.Bool(true),
//                     OrderBy:      types.OrderByLastEventTime,
//                 }

//                 logStreams, err := s.client.DescribeLogStreams(ctx, input)
//                 if err != nil {
//                     log.Printf("Error describing log streams: %v", err)
//                     continue
//                 }

//                 for _, stream := range logStreams.LogStreams {
//                     if !strings.Contains(strings.ToLower(*stream.LogStreamName), strings.ToLower(appName)) {
//                         continue
//                     }

//                     params := &cloudwatchlogs.GetLogEventsInput{
//                         LogGroupName:  aws.String("/aws/containerinsights/paas-1/application"),
//                         LogStreamName: stream.LogStreamName,
//                         StartTime:     aws.Int64(latestTimestamp),
//                         StartFromHead: aws.Bool(false),
//                         Limit:         aws.Int32(100),
//                     }

//                     output, err := s.client.GetLogEvents(ctx, params)
//                     if err != nil {
//                         log.Printf("Error getting log events: %v", err)
//                         continue
//                     }

//                     for _, event := range output.Events {
//                         // Create a unique identifier for each log event
//                         logID := fmt.Sprintf("%d-%s", *event.Timestamp, *event.Message)
                        
//                         // Only process if we haven't seen this log before and it's newer than our latest timestamp
//                         if !seenLogs[logID] && *event.Timestamp > latestTimestamp {
//                             seenLogs[logID] = true
//                             if *event.Timestamp > latestTimestamp {
//                                 latestTimestamp = *event.Timestamp
//                             }
                            
//                             select {
//                             case logChan <- *event.Message:
//                             case <-ctx.Done():
//                                 return
//                             }
//                         }
//                     }
//                 }
//             }

//             // Wait before next poll
//             select {
//             case <-ctx.Done():
//                 return
//             case <-time.After(5 * time.Second):
//                 // Clean up old entries periodically
//                 if len(seenLogs) > 1000 {
//                     seenLogs = make(map[string]bool)
//                 }
//             }
//         }
//     }()

//     return logChan, errChan
// }


package services

import (
    "context"
    "fmt"
    "time"
    "strings"
    "log"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type LogFilter struct {
    SearchText  string    
    StartTime   time.Time 
    EndTime     time.Time 
}

type LogService interface {
    StreamLogs(ctx context.Context, appName string, filter LogFilter) (<-chan string, <-chan error)
}

type LogServiceImpl struct {
    client *cloudwatchlogs.Client
}

func NewLogService(ctx context.Context) (LogService, error) {
    log.Printf("Initializing LogService...")
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Printf("ERROR: Failed to load AWS config: %v", err)
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := cloudwatchlogs.NewFromConfig(cfg)
    return &LogServiceImpl{client: client}, nil
}

func (s *LogServiceImpl) StreamLogs(ctx context.Context, appName string, filter LogFilter) (<-chan string, <-chan error) {
    logChan := make(chan string)
    errChan := make(chan error, 1)
    
    log.Printf("Starting to stream logs for app: %s with filters - SearchText: %s, TimeRange: %v to %v", 
        appName, filter.SearchText, filter.StartTime, filter.EndTime)

    go func() {
        defer close(logChan)
        defer close(errChan)

        // Keep track of the latest timestamp we've seen
        var latestTimestamp int64 = filter.StartTime.UnixNano() / int64(time.Millisecond)
        seenLogs := make(map[string]bool)

        for {
            select {
            case <-ctx.Done():
                log.Printf("Context cancelled, stopping log streaming")
                return
            default:
                // Try different possible log group names
                logGroups := []string{
                    "/aws/containerinsights/paas-1/application",
                    fmt.Sprintf("/aws/eks/paas-1/cluster"),
                    fmt.Sprintf("/eks/paas-1/containers"),
                }

                for _, logGroupName := range logGroups {
                    log.Printf("Checking log group: %s", logGroupName)

                    input := &cloudwatchlogs.DescribeLogStreamsInput{
                        LogGroupName: aws.String(logGroupName),
                        Descending:   aws.Bool(true),
                        OrderBy:      types.OrderByLastEventTime,
                    }

                    logStreams, err := s.client.DescribeLogStreams(ctx, input)
                    if err != nil {
                        log.Printf("Error describing log streams for group %s: %v", logGroupName, err)
                        continue
                    }

                    log.Printf("Found %d streams in group %s", len(logStreams.LogStreams), logGroupName)

                    // Find relevant streams
                    for _, stream := range logStreams.LogStreams {
                        streamName := *stream.LogStreamName
                        if !strings.Contains(strings.ToLower(streamName), strings.ToLower(appName)) {
                            continue
                        }

                        log.Printf("Checking stream: %s for new logs", streamName)
                        
                        params := &cloudwatchlogs.GetLogEventsInput{
                            LogGroupName:  aws.String(logGroupName),
                            LogStreamName: aws.String(streamName),
                            StartTime:     aws.Int64(latestTimestamp),
                            StartFromHead: aws.Bool(false),
                            Limit:         aws.Int32(100),
                        }

                        var lastToken *string
                        newEvents := false

                        for {
                            if lastToken != nil && params.NextToken != nil && *lastToken == *params.NextToken {
                                break
                            }

                            output, err := s.client.GetLogEvents(ctx, params)
                            if err != nil {
                                log.Printf("Error getting log events: %v", err)
                                break
                            }

                            for _, event := range output.Events {
                                // Check time range
                                eventTime := time.Unix(0, *event.Timestamp*int64(time.Millisecond))
                                if eventTime.After(filter.EndTime) {
                                    continue
                                }

                                // Apply text filter if specified
                                if filter.SearchText != "" && !strings.Contains(
                                    strings.ToLower(*event.Message),
                                    strings.ToLower(filter.SearchText),
                                ) {
                                    continue
                                }

                                // Create a unique identifier for each log event
                                logID := fmt.Sprintf("%d-%s", *event.Timestamp, *event.Message)
                                
                                // Only process if we haven't seen this log before and it's newer than our latest timestamp
                                if !seenLogs[logID] && *event.Timestamp > latestTimestamp {
                                    newEvents = true
                                    seenLogs[logID] = true
                                    if *event.Timestamp > latestTimestamp {
                                        latestTimestamp = *event.Timestamp
                                    }
                                    
                                    log.Printf("New event at %v: %s", eventTime, *event.Message)
                                    
                                    select {
                                    case logChan <- *event.Message:
                                    case <-ctx.Done():
                                        return
                                    }
                                }
                            }

                            lastToken = params.NextToken
                            params.NextToken = output.NextForwardToken

                            if !newEvents {
                                log.Printf("No new events found in stream %s", streamName)
                            }
                        }
                    }
                }

                // Wait before checking for new logs
                select {
                case <-ctx.Done():
                    return
                case <-time.After(5 * time.Second):
                    // Clean up old entries periodically
                    if len(seenLogs) > 1000 {
                        log.Printf("Cleaning up old log entries from memory")
                        seenLogs = make(map[string]bool)
                    }
                    log.Printf("Checking for new logs...")
                }
            }
        }
    }()

    return logChan, errChan
}