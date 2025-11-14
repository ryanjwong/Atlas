package logsource

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type AWSLogSource struct {
	profile string
	region  string
}

func NewAWSLogSource(profile, region string) *AWSLogSource {
	return &AWSLogSource{
		profile: profile,
		region:  region,
	}
}

func (a *AWSLogSource) GetSourceName() string {
	return "aws"
}

func (a *AWSLogSource) GetClusterHistory(ctx context.Context, clusterName string, limit int) ([]*OperationHistory, error) {
	cmd := exec.CommandContext(ctx, "aws", "logs", "describe-log-streams",
		"--log-group-name", fmt.Sprintf("/aws/eks/%s/cluster", clusterName),
		"--region", a.region,
		"--max-items", fmt.Sprintf("%d", limit))

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return []*OperationHistory{}, nil
	}

	var logStreams struct {
		LogStreams []struct {
			LogStreamName    string `json:"logStreamName"`
			CreationTime     int64  `json:"creationTime"`
			LastEventTime    int64  `json:"lastEventTime"`
			LastIngestionTime int64 `json:"lastIngestionTime"`
		} `json:"logStreams"`
	}

	if err := json.Unmarshal(output, &logStreams); err != nil {
		return nil, fmt.Errorf("failed to parse log streams: %w", err)
	}

	var history []*OperationHistory
	for _, stream := range logStreams.LogStreams {
		op := &OperationHistory{
			ClusterName:     clusterName,
			OperationType:   OpTypeUpdate,
			OperationStatus: OpStatusCompleted,
			StartedAt:       time.Unix(stream.CreationTime/1000, 0),
			UserID:          "aws-system",
			OperationDetails: map[string]interface{}{
				"log_stream": stream.LogStreamName,
			},
			Metadata: map[string]string{
				"source": "aws-cloudwatch",
				"region": a.region,
			},
		}

		if stream.LastEventTime > 0 {
			completedAt := time.Unix(stream.LastEventTime/1000, 0)
			op.CompletedAt = &completedAt
			duration := completedAt.Sub(op.StartedAt)
			durationMS := float64(duration.Milliseconds())
			op.DurationMS = &durationMS
		}

		history = append(history, op)
	}

	return history, nil
}

func (a *AWSLogSource) GetAllClustersHistory(ctx context.Context, limit int) (map[string][]*OperationHistory, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "list-clusters",
		"--region", a.region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var result struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cluster list: %w", err)
	}

	clusterHistories := make(map[string][]*OperationHistory)

	for _, clusterName := range result.Clusters {
		history, err := a.GetClusterHistory(ctx, clusterName, limit/len(result.Clusters))
		if err != nil {
			continue
		}
		clusterHistories[clusterName] = history
	}

	return clusterHistories, nil
}

func (a *AWSLogSource) getClusterEvents(ctx context.Context, clusterName string) ([]*OperationHistory, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster",
		"--name", clusterName,
		"--region", a.region,
		"--query", "cluster.{name:name,status:status,createdAt:createdAt,version:version}")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	var clusterData struct {
		Name      string    `json:"name"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"createdAt"`
		Version   string    `json:"version"`
	}

	if err := json.Unmarshal(output, &clusterData); err != nil {
		return nil, fmt.Errorf("failed to parse cluster data: %w", err)
	}

	var operations []*OperationHistory

	createOp := &OperationHistory{
		ClusterName:     clusterData.Name,
		OperationType:   OpTypeCreate,
		OperationStatus: OpStatusCompleted,
		StartedAt:       clusterData.CreatedAt,
		UserID:          "aws-user",
		OperationDetails: map[string]interface{}{
			"kubernetes_version": clusterData.Version,
			"status":            clusterData.Status,
		},
		Metadata: map[string]string{
			"source":   "aws-eks",
			"region":   a.region,
			"provider": "aws",
		},
	}

	if strings.ToLower(clusterData.Status) == "active" {
		completedAt := clusterData.CreatedAt.Add(10 * time.Minute)
		createOp.CompletedAt = &completedAt
		duration := completedAt.Sub(clusterData.CreatedAt)
		durationMS := float64(duration.Milliseconds())
		createOp.DurationMS = &durationMS
	}

	operations = append(operations, createOp)

	return operations, nil
}