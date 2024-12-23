package services

// import (
// 	"context"
// 	"fmt"

// 	"px.dev/pxapi"
// 	"px.dev/pxapi/types"
// )

// // GetNamespaceMetrics fetches metrics for a specific Kubernetes namespace.
// func GetNamespaceMetrics(namespace string) ([][]interface{}, error) {
// 	ctx := context.Background()
// 	apiKey := "YOUR_PIXIE_API_KEY"      // Replace with your Pixie API Key
// 	clusterID := "YOUR_CLUSTER_ID"      // Replace with your Cluster ID

// 	// Create a Pixie client.
// 	client, err := pxapi.NewClient(ctx, pxapi.WithAPIKey(apiKey))
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Connect to the cluster.
// 	vz, err := client.NewVizierClient(ctx, clusterID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Define a PxL script to fetch metrics for the given namespace.
// 	script := fmt.Sprintf(`
// 		import px
// 		df = px.DataFrame('http_events', namespace='%s')
// 		df = df[['time_', 'service', 'req_method', 'resp_status', 'latency']]
// 		px.display(df, 'namespace_metrics')
// 	`, namespace)

// 	// Create a TableMuxer to accept the results.
// 	tm := &tableMuxer{}
// 	resultSet, err := vz.ExecuteScript(ctx, script, tm)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resultSet.Close()

// 	// Stream the results.
// 	if err := resultSet.Stream(); err != nil {
// 		return nil, err
// 	}

// 	return tm.records, nil
// }

// // tableMuxer handles the table data received from Pixie.
// type tableMuxer struct {
// 	records [][]interface{}
// }

// // AcceptTable initializes the table record handler.
// func (t *tableMuxer) AcceptTable(ctx context.Context, metadata types.TableMetadata) (pxapi.TableRecordHandler, error) {
// 	return &tableRecordHandler{records: &t.records}, nil
// }

// // tableRecordHandler processes each record in the table.
// type tableRecordHandler struct {
// 	records *[][]interface{}
// }

// // HandleInit performs any initialization before processing records.
// func (t *tableRecordHandler) HandleInit(ctx context.Context, metadata types.TableMetadata) error {
// 	return nil
// }

// // HandleRecord processes each record and stores it.
// func (t *tableRecordHandler) HandleRecord(ctx context.Context, r *types.Record) error {
// 	row := make([]interface{}, len(r.Data))
// 	for i, d := range r.Data {
// 		row[i] = d.Value()
// 	}
// 	*t.records = append(*t.records, row)
// 	return nil
// }

// // HandleDone performs any cleanup after all records are processed.
// func (t *tableRecordHandler) HandleDone(ctx context.Context) error {
// 	return nil
// }