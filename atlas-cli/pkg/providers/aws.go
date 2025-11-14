package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/logsource"
	"github.com/ryanjwong/Atlas/atlas-cli/pkg/monitoring"
)

type AWSProvider struct {
	profile   string
	region    string
	logSource logsource.LogSource
	monitor   monitoring.Monitor
}

type EKSCluster struct {
	Name     string            `json:"name"`
	Arn      string            `json:"arn"`
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Endpoint string            `json:"endpoint"`
	Tags     map[string]string `json:"tags"`
	CreatedAt time.Time        `json:"createdAt"`
}

type EKSNodegroup struct {
	NodegroupName string            `json:"nodegroupName"`
	Status        string            `json:"status"`
	InstanceTypes []string          `json:"instanceTypes"`
	AmiType       string            `json:"amiType"`
	NodeRole      string            `json:"nodeRole"`
	Subnets       []string          `json:"subnets"`
	RemoteAccess  map[string]interface{} `json:"remoteAccess"`
	ScalingConfig EKSScalingConfig  `json:"scalingConfig"`
	Tags          map[string]string `json:"tags"`
	CreatedAt     time.Time         `json:"createdAt"`
	ModifiedAt    time.Time         `json:"modifiedAt"`
}

type EKSScalingConfig struct {
	MinSize     int `json:"minSize"`
	MaxSize     int `json:"maxSize"`
	DesiredSize int `json:"desiredSize"`
}

func NewAWSProvider(profile, region string) *AWSProvider {
	return &AWSProvider{
		profile:   profile,
		region:    region,
		logSource: logsource.NewAWSLogSource(profile, region),
		monitor:   monitoring.NewAWSMonitor(profile, region),
	}
}

func (a *AWSProvider) GetProviderName() string {
	return "aws"
}

func (a *AWSProvider) GetSupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
		"ca-central-1", "sa-east-1",
	}
}

func (a *AWSProvider) GetSupportedVersions() []string {
	versions, err := a.getEKSVersions()
	if err != nil {
		return []string{"1.31", "1.30", "1.29", "1.28", "1.27"}
	}
	return versions
}

func (a *AWSProvider) GetLogSource() logsource.LogSource {
	return a.logSource
}

func (a *AWSProvider) GetMonitor() monitoring.Monitor {
	return a.monitor
}

func (a *AWSProvider) HealthCheck(ctx context.Context, clusterName string) (*monitoring.HealthStatus, error) {
	return a.monitor.CheckClusterHealth(ctx, clusterName)
}

func (a *AWSProvider) ValidateConfig(config *ClusterConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	if config.Region != "" {
		validRegions := a.GetSupportedRegions()
		regionValid := false
		for _, region := range validRegions {
			if region == config.Region {
				regionValid = true
				break
			}
		}
		if !regionValid {
			return fmt.Errorf("unsupported region: %s", config.Region)
		}
	}

	if config.Version != "" {
		supportedVersions := a.GetSupportedVersions()
		versionValid := false
		for _, version := range supportedVersions {
			if version == config.Version {
				versionValid = true
				break
			}
		}
		if !versionValid {
			return fmt.Errorf("unsupported EKS version: %s", config.Version)
		}
	}

	if config.NodeCount < 1 {
		return fmt.Errorf("node count must be at least 1")
	}

	if config.NodeCount > 100 {
		return fmt.Errorf("node count cannot exceed 100 for EKS")
	}

	if config.InstanceType != "" {
		validInstanceTypes := []string{
			"t3.micro", "t3.small", "t3.medium", "t3.large", "t3.xlarge", "t3.2xlarge",
			"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m5.16xlarge", "m5.24xlarge",
			"c5.large", "c5.xlarge", "c5.2xlarge", "c5.4xlarge", "c5.9xlarge", "c5.12xlarge", "c5.18xlarge", "c5.24xlarge",
			"r5.large", "r5.xlarge", "r5.2xlarge", "r5.4xlarge", "r5.8xlarge", "r5.12xlarge", "r5.16xlarge", "r5.24xlarge",
		}
		instanceValid := false
		for _, instance := range validInstanceTypes {
			if instance == config.InstanceType {
				instanceValid = true
				break
			}
		}
		if !instanceValid {
			return fmt.Errorf("unsupported instance type: %s", config.InstanceType)
		}
	}

	return nil
}

func (a *AWSProvider) CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error) {
	if err := a.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	region := config.Region
	if region == "" {
		region = a.region
	}

	version := config.Version
	if version == "" {
		version = "1.31"
	}

	cmd := exec.CommandContext(ctx, "aws", "eks", "create-cluster",
		"--name", config.Name,
		"--version", version,
		"--role-arn", a.getClusterServiceRoleArn(),
		"--resources-vpc-config", a.buildVpcConfig(config),
		"--region", region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS cluster: %s", string(output))
	}

	var result struct {
		Cluster EKSCluster `json:"cluster"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create cluster response: %w", err)
	}

	if err := a.waitForClusterActive(ctx, config.Name, region); err != nil {
		return nil, fmt.Errorf("cluster creation failed: %w", err)
	}

	if err := a.createNodeGroup(ctx, config, region); err != nil {
		return nil, fmt.Errorf("failed to create node group: %w", err)
	}

	return a.GetCluster(ctx, config.Name)
}

func (a *AWSProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster",
		"--name", name,
		"--region", a.region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	var result struct {
		Cluster EKSCluster `json:"cluster"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cluster description: %w", err)
	}

	nodeCount, err := a.getNodeCount(ctx, name)
	if err != nil {
		nodeCount = 0
	}

	status := ClusterStatusRunning
	switch strings.ToLower(result.Cluster.Status) {
	case "creating":
		status = ClusterStatusPending
	case "active":
		status = ClusterStatusRunning
	case "deleting":
		status = ClusterStatusDeleting
	case "failed":
		status = ClusterStatusError
	default:
		status = ClusterStatusError
	}

	return &Cluster{
		Name:      result.Cluster.Name,
		Provider:  "aws",
		Region:    a.region,
		Version:   result.Cluster.Version,
		Status:    status,
		NodeCount: nodeCount,
		Endpoint:  result.Cluster.Endpoint,
		CreatedAt: result.Cluster.CreatedAt,
		UpdatedAt: time.Now(),
		Tags:      result.Cluster.Tags,
	}, nil
}

func (a *AWSProvider) ListClusters(ctx context.Context) ([]*Cluster, error) {
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

	var clusters []*Cluster
	for _, clusterName := range result.Clusters {
		cluster, err := a.GetCluster(ctx, clusterName)
		if err != nil {
			continue
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (a *AWSProvider) DeleteCluster(ctx context.Context, name string) error {
	if err := a.deleteNodeGroups(ctx, name); err != nil {
		return fmt.Errorf("failed to delete node groups: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "eks", "delete-cluster",
		"--name", name,
		"--region", a.region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %s", string(output))
	}

	return nil
}

func (a *AWSProvider) StartCluster(ctx context.Context, name string) error {
	return fmt.Errorf("EKS clusters cannot be started/stopped - they are always running once created")
}

func (a *AWSProvider) StopCluster(ctx context.Context, name string) error {
	return fmt.Errorf("EKS clusters cannot be started/stopped - they are always running once created")
}

func (a *AWSProvider) ScaleCluster(ctx context.Context, name string, nodeCount int) error {
	if nodeCount < 1 {
		return fmt.Errorf("node count must be at least 1")
	}

	nodeGroups, err := a.listNodeGroups(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to list node groups: %w", err)
	}

	if len(nodeGroups) == 0 {
		return fmt.Errorf("no node groups found for cluster %s", name)
	}

	nodeGroupName := nodeGroups[0]

	cmd := exec.CommandContext(ctx, "aws", "eks", "update-nodegroup-config",
		"--cluster-name", name,
		"--nodegroup-name", nodeGroupName,
		"--scaling-config", fmt.Sprintf("minSize=1,maxSize=%d,desiredSize=%d", nodeCount, nodeCount),
		"--region", a.region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to scale cluster: %s", string(output))
	}

	return nil
}

func (a *AWSProvider) getEKSVersions() ([]string, error) {
	cmd := exec.Command("aws", "eks", "describe-addon-versions",
		"--kubernetes-version", "1.31",
		"--region", a.region,
		"--query", "addons[0].addonVersions[0].compatibilities[*].clusterVersion",
		"--output", "json")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var versions []string
	if err := json.Unmarshal(output, &versions); err != nil {
		return nil, err
	}

	return versions, nil
}

func (a *AWSProvider) getClusterServiceRoleArn() string {
	return fmt.Sprintf("arn:aws:iam::%s:role/eks-service-role", a.getAccountID())
}

func (a *AWSProvider) getNodeInstanceRoleArn() string {
	return fmt.Sprintf("arn:aws:iam::%s:role/NodeInstanceRole", a.getAccountID())
}

func (a *AWSProvider) getAccountID() string {
	cmd := exec.Command("aws", "sts", "get-caller-identity",
		"--query", "Account",
		"--output", "text")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return "123456789012"
	}

	return strings.TrimSpace(string(output))
}

func (a *AWSProvider) buildVpcConfig(config *ClusterConfig) string {
	return "subnetIds=subnet-12345,subnet-67890,endpointConfigAccess={publicAccess=true,privateAccess=true}"
}

func (a *AWSProvider) waitForClusterActive(ctx context.Context, name, region string) error {
	maxWait := 20 * time.Minute
	checkInterval := 30 * time.Second
	
	deadline := time.Now().Add(maxWait)
	
	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster",
			"--name", name,
			"--region", region,
			"--query", "cluster.status",
			"--output", "text")

		if a.profile != "" {
			cmd.Args = append(cmd.Args, "--profile", a.profile)
		}

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check cluster status: %w", err)
		}

		status := strings.TrimSpace(string(output))
		if status == "ACTIVE" {
			return nil
		}

		if status == "FAILED" {
			return fmt.Errorf("cluster creation failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
		}
	}

	return fmt.Errorf("timeout waiting for cluster to become active")
}

func (a *AWSProvider) createNodeGroup(ctx context.Context, config *ClusterConfig, region string) error {
	instanceType := config.InstanceType
	if instanceType == "" {
		instanceType = "t3.medium"
	}

	cmd := exec.CommandContext(ctx, "aws", "eks", "create-nodegroup",
		"--cluster-name", config.Name,
		"--nodegroup-name", fmt.Sprintf("%s-nodes", config.Name),
		"--subnets", "subnet-12345,subnet-67890",
		"--node-role", a.getNodeInstanceRoleArn(),
		"--instance-types", instanceType,
		"--scaling-config", fmt.Sprintf("minSize=1,maxSize=%d,desiredSize=%d", config.NodeCount, config.NodeCount),
		"--region", region)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create node group: %s", string(output))
	}

	return a.waitForNodeGroupActive(ctx, config.Name, fmt.Sprintf("%s-nodes", config.Name), region)
}

func (a *AWSProvider) waitForNodeGroupActive(ctx context.Context, clusterName, nodeGroupName, region string) error {
	maxWait := 15 * time.Minute
	checkInterval := 30 * time.Second
	
	deadline := time.Now().Add(maxWait)
	
	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "aws", "eks", "describe-nodegroup",
			"--cluster-name", clusterName,
			"--nodegroup-name", nodeGroupName,
			"--region", region,
			"--query", "nodegroup.status",
			"--output", "text")

		if a.profile != "" {
			cmd.Args = append(cmd.Args, "--profile", a.profile)
		}

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check node group status: %w", err)
		}

		status := strings.TrimSpace(string(output))
		if status == "ACTIVE" {
			return nil
		}

		if status == "CREATE_FAILED" {
			return fmt.Errorf("node group creation failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
		}
	}

	return fmt.Errorf("timeout waiting for node group to become active")
}

func (a *AWSProvider) getNodeCount(ctx context.Context, clusterName string) (int, error) {
	nodeGroups, err := a.listNodeGroups(ctx, clusterName)
	if err != nil {
		return 0, err
	}

	totalNodes := 0
	for _, nodeGroupName := range nodeGroups {
		cmd := exec.CommandContext(ctx, "aws", "eks", "describe-nodegroup",
			"--cluster-name", clusterName,
			"--nodegroup-name", nodeGroupName,
			"--region", a.region,
			"--query", "nodegroup.scalingConfig.desiredSize",
			"--output", "text")

		if a.profile != "" {
			cmd.Args = append(cmd.Args, "--profile", a.profile)
		}

		output, err := cmd.Output()
		if err != nil {
			continue
		}

		var desiredSize int
		fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &desiredSize)
		totalNodes += desiredSize
	}

	return totalNodes, nil
}

func (a *AWSProvider) listNodeGroups(ctx context.Context, clusterName string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "list-nodegroups",
		"--cluster-name", clusterName,
		"--region", a.region,
		"--query", "nodegroups",
		"--output", "json")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list node groups: %w", err)
	}

	var nodeGroups []string
	if err := json.Unmarshal(output, &nodeGroups); err != nil {
		return nil, fmt.Errorf("failed to parse node groups: %w", err)
	}

	return nodeGroups, nil
}

func (a *AWSProvider) deleteNodeGroups(ctx context.Context, clusterName string) error {
	nodeGroups, err := a.listNodeGroups(ctx, clusterName)
	if err != nil {
		return err
	}

	for _, nodeGroupName := range nodeGroups {
		cmd := exec.CommandContext(ctx, "aws", "eks", "delete-nodegroup",
			"--cluster-name", clusterName,
			"--nodegroup-name", nodeGroupName,
			"--region", a.region)

		if a.profile != "" {
			cmd.Args = append(cmd.Args, "--profile", a.profile)
		}

		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete node group %s: %w", nodeGroupName, err)
		}
	}

	return nil
}

var _ Provider = (*AWSProvider)(nil)