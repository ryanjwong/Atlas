package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type AWSMonitor struct {
	profile            string
	region             string
	activeMonitoring   map[string]context.CancelFunc
}

func NewAWSMonitor(profile, region string) *AWSMonitor {
	return &AWSMonitor{
		profile:          profile,
		region:           region,
		activeMonitoring: make(map[string]context.CancelFunc),
	}
}

func (a *AWSMonitor) GetMonitorName() string {
	return "aws"
}

func (a *AWSMonitor) CheckClusterHealth(ctx context.Context, clusterName string) (*HealthStatus, error) {
	startTime := time.Now()
	
	status := &HealthStatus{
		ClusterName:   clusterName,
		OverallStatus: HealthStatusUnknown,
		LastChecked:   startTime,
		Warnings:      []string{},
		Errors:        []string{},
	}

	clusterStatus, err := a.getEKSClusterStatus(ctx, clusterName)
	if err != nil {
		status.OverallStatus = HealthStatusUnhealthy
		status.Errors = append(status.Errors, fmt.Sprintf("Failed to get cluster status: %v", err))
		status.CheckDuration = time.Since(startTime)
		return status, nil
	}

	if strings.ToLower(clusterStatus) != "active" {
		status.OverallStatus = HealthStatusUnhealthy
		status.Errors = append(status.Errors, fmt.Sprintf("Cluster status is %s, expected ACTIVE", clusterStatus))
		status.CheckDuration = time.Since(startTime)
		return status, nil
	}

	controlPlaneHealth, err := a.checkControlPlane(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Control plane check failed: %v", err))
	} else {
		status.ControlPlane = controlPlaneHealth
	}

	nodes, err := a.checkNodes(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Node check failed: %v", err))
	} else {
		status.Nodes = nodes
	}

	podHealth, err := a.checkPods(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Pod check failed: %v", err))
	} else {
		status.Pods = podHealth
	}

	serviceHealth, err := a.checkServices(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Service check failed: %v", err))
	} else {
		status.Services = serviceHealth
	}

	status.OverallStatus = a.calculateOverallHealth(status)
	status.CheckDuration = time.Since(startTime)

	return status, nil
}

func (a *AWSMonitor) GetClusterMetrics(ctx context.Context, clusterName string) (*ClusterMetrics, error) {
	timestamp := time.Now()
	
	metrics := &ClusterMetrics{
		ClusterName: clusterName,
		Timestamp:   timestamp,
	}

	if !a.isEKSClusterActive(ctx, clusterName) {
		return nil, fmt.Errorf("cluster %s is not active", clusterName)
	}

	nodeMetrics, err := a.getNodeMetrics(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}
	metrics.NodeMetrics = nodeMetrics

	podMetrics, err := a.getPodMetrics(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}
	metrics.PodMetrics = podMetrics

	resourceUsage, err := a.calculateResourceUsage(nodeMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate resource usage: %w", err)
	}
	metrics.ResourceUsage = resourceUsage

	return metrics, nil
}

func (a *AWSMonitor) StartMonitoring(ctx context.Context, config *MonitoringConfig) error {
	for _, clusterName := range config.ClusterNames {
		if _, exists := a.activeMonitoring[clusterName]; exists {
			continue
		}
		
		monitorCtx, cancel := context.WithCancel(ctx)
		a.activeMonitoring[clusterName] = cancel
		
		go a.monitorCluster(monitorCtx, clusterName, config)
	}
	
	return nil
}

func (a *AWSMonitor) StopMonitoring(ctx context.Context, clusterName string) error {
	if cancel, exists := a.activeMonitoring[clusterName]; exists {
		cancel()
		delete(a.activeMonitoring, clusterName)
	}
	
	return nil
}

func (a *AWSMonitor) monitorCluster(ctx context.Context, clusterName string, config *MonitoringConfig) {
	healthTicker := time.NewTicker(config.CheckInterval)
	metricsTicker := time.NewTicker(config.MetricsInterval)
	
	defer healthTicker.Stop()
	defer metricsTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-healthTicker.C:
			_, err := a.CheckClusterHealth(ctx, clusterName)
			if err != nil && config.EnableAlerts {
				fmt.Printf("Health check failed for EKS cluster %s: %v\n", clusterName, err)
			}
		case <-metricsTicker.C:
			_, err := a.GetClusterMetrics(ctx, clusterName)
			if err != nil && config.EnableAlerts {
				fmt.Printf("Metrics collection failed for EKS cluster %s: %v\n", clusterName, err)
			}
		}
	}
}

func (a *AWSMonitor) getEKSClusterStatus(ctx context.Context, clusterName string) (string, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster",
		"--name", clusterName,
		"--region", a.region,
		"--query", "cluster.status",
		"--output", "text")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster status: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (a *AWSMonitor) isEKSClusterActive(ctx context.Context, clusterName string) bool {
	status, err := a.getEKSClusterStatus(ctx, clusterName)
	if err != nil {
		return false
	}
	return strings.ToLower(status) == "active"
}

func (a *AWSMonitor) checkControlPlane(ctx context.Context, clusterName string) (*ControlPlaneHealth, error) {
	cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster",
		"--name", clusterName,
		"--region", a.region,
		"--query", "cluster.{endpoint:endpoint,version:version,status:status}")

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster details: %w", err)
	}

	var clusterInfo struct {
		Endpoint string `json:"endpoint"`
		Version  string `json:"version"`
		Status   string `json:"status"`
	}

	if err := json.Unmarshal(output, &clusterInfo); err != nil {
		return nil, fmt.Errorf("failed to parse cluster details: %w", err)
	}

	health := &ControlPlaneHealth{
		APIServer:         ComponentStatus{Status: ComponentHealthy, LastCheck: time.Now()},
		Scheduler:         ComponentStatus{Status: ComponentHealthy, LastCheck: time.Now()},
		ControllerManager: ComponentStatus{Status: ComponentHealthy, LastCheck: time.Now()},
		Etcd:              ComponentStatus{Status: ComponentHealthy, LastCheck: time.Now()},
	}

	if strings.ToLower(clusterInfo.Status) != "active" {
		health.APIServer.Status = ComponentUnhealthy
		health.APIServer.Message = fmt.Sprintf("Cluster status: %s", clusterInfo.Status)
	}

	return health, nil
}

func (a *AWSMonitor) checkNodes(ctx context.Context, clusterName string) ([]NodeHealth, error) {
	if err := a.updateKubeConfig(ctx, clusterName); err != nil {
		return nil, fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "json", "--context", fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", a.region, a.getAccountID(), clusterName))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var nodeList struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct {
					Type               string `json:"type"`
					Status             string `json:"status"`
					LastTransitionTime string `json:"lastTransitionTime"`
					Reason             string `json:"reason,omitempty"`
					Message            string `json:"message,omitempty"`
				} `json:"conditions"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
				Capacity struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"capacity"`
				Allocatable struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"allocatable"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to parse node list: %w", err)
	}

	var nodes []NodeHealth
	for _, node := range nodeList.Items {
		nodeHealth := NodeHealth{
			Name:        node.Metadata.Name,
			Status:      NodeUnknown,
			Ready:       false,
			Version:     node.Status.NodeInfo.KubeletVersion,
			LastChecked: time.Now(),
			Resources: &NodeResources{
				CPUCapacity:       node.Status.Capacity.CPU,
				MemoryCapacity:    node.Status.Capacity.Memory,
				CPUAllocatable:    node.Status.Allocatable.CPU,
				MemoryAllocatable: node.Status.Allocatable.Memory,
			},
		}

		for _, condition := range node.Status.Conditions {
			conditionTime, _ := time.Parse(time.RFC3339, condition.LastTransitionTime)

			nodeCondition := NodeCondition{
				Type:               condition.Type,
				Status:             condition.Status,
				LastTransitionTime: conditionTime,
				Reason:             condition.Reason,
				Message:            condition.Message,
			}
			nodeHealth.Conditions = append(nodeHealth.Conditions, nodeCondition)

			if condition.Type == "Ready" {
				if condition.Status == "True" {
					nodeHealth.Ready = true
					nodeHealth.Status = NodeHealthy
				} else {
					nodeHealth.Status = NodeNotReady
				}
			}
		}

		nodes = append(nodes, nodeHealth)
	}

	return nodes, nil
}

func (a *AWSMonitor) checkPods(ctx context.Context, clusterName string) (*PodHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "--all-namespaces", "-o", "json", "--context", fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", a.region, a.getAccountID(), clusterName))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					RestartCount int `json:"restartCount"`
				} `json:"containerStatuses,omitempty"`
				Message string `json:"message,omitempty"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return nil, fmt.Errorf("failed to parse pod list: %w", err)
	}

	podHealth := &PodHealth{
		PodsByPhase: make(map[string]int),
		Namespaces:  make(map[string]*NamespaceHealth),
	}

	for _, pod := range podList.Items {
		phase := pod.Status.Phase
		podHealth.TotalPods++
		podHealth.PodsByPhase[phase]++

		switch phase {
		case "Running":
			podHealth.RunningPods++
		case "Pending":
			podHealth.PendingPods++
		case "Failed":
			podHealth.FailedPods++
		case "Succeeded":
			podHealth.SucceededPods++
		default:
			podHealth.UnknownPods++
		}

		if podHealth.Namespaces[pod.Metadata.Namespace] == nil {
			podHealth.Namespaces[pod.Metadata.Namespace] = &NamespaceHealth{
				Name:   pod.Metadata.Namespace,
				Status: "healthy",
			}
		}
		ns := podHealth.Namespaces[pod.Metadata.Namespace]
		ns.TotalPods++
		if phase == "Running" {
			ns.HealthyPods++
		}

		if phase == "Failed" || (len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].RestartCount > 5) {
			criticalPod := CriticalPodInfo{
				Name:      pod.Metadata.Name,
				Namespace: pod.Metadata.Namespace,
				Phase:     phase,
				Message:   pod.Status.Message,
			}
			if len(pod.Status.ContainerStatuses) > 0 {
				criticalPod.RestartCount = pod.Status.ContainerStatuses[0].RestartCount
			}
			podHealth.CriticalPods = append(podHealth.CriticalPods, criticalPod)
		}
	}

	return podHealth, nil
}

func (a *AWSMonitor) checkServices(ctx context.Context, clusterName string) (*ServiceHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "services", "--all-namespaces", "-o", "json", "--context", fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", a.region, a.getAccountID(), clusterName))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	var serviceList struct {
		Items []struct {
			Spec struct {
				Type string `json:"type"`
			} `json:"spec"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &serviceList); err != nil {
		return nil, fmt.Errorf("failed to parse service list: %w", err)
	}

	serviceHealth := &ServiceHealth{
		ServicesByType: make(map[string]int),
	}

	for _, service := range serviceList.Items {
		serviceHealth.TotalServices++
		serviceHealth.HealthyServices++
		serviceHealth.ServicesByType[service.Spec.Type]++
	}

	return serviceHealth, nil
}

func (a *AWSMonitor) getNodeMetrics(ctx context.Context, clusterName string) ([]NodeMetrics, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "nodes", "--context", fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", a.region, a.getAccountID(), clusterName), "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics (metrics server may not be installed): %w", err)
	}

	var metrics []NodeMetrics
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		nodeName := fields[0]
		cpuUsage := fields[1]
		memUsage := fields[3]

		metrics = append(metrics, NodeMetrics{
			NodeName: nodeName,
			CPUUsage: ResourceValue{
				Value: cpuUsage,
			},
			MemoryUsage: ResourceValue{
				Value: memUsage,
			},
			Timestamp: time.Now(),
		})
	}

	return metrics, nil
}

func (a *AWSMonitor) getPodMetrics(ctx context.Context, clusterName string) ([]PodMetrics, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "pods", "--all-namespaces", "--context", fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", a.region, a.getAccountID(), clusterName), "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics (metrics server may not be installed): %w", err)
	}

	var metrics []PodMetrics
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		namespace := fields[0]
		podName := fields[1]
		cpuUsage := fields[2]
		memUsage := fields[3]

		metrics = append(metrics, PodMetrics{
			PodName:   podName,
			Namespace: namespace,
			CPUUsage: ResourceValue{
				Value: cpuUsage,
			},
			MemoryUsage: ResourceValue{
				Value: memUsage,
			},
			Containers: make(map[string]ContainerMetrics),
			Timestamp:  time.Now(),
		})
	}

	return metrics, nil
}

func (a *AWSMonitor) calculateResourceUsage(nodeMetrics []NodeMetrics) (*ResourceUsage, error) {
	if len(nodeMetrics) == 0 {
		return &ResourceUsage{}, nil
	}

	totalCPUUsage := 0.0
	totalMemUsage := 0.0
	cpuUsageStr := ""
	memUsageStr := ""

	for _, node := range nodeMetrics {
		totalCPUUsage += node.CPUUsage.Usage
		totalMemUsage += node.MemoryUsage.Usage

		if cpuUsageStr == "" {
			cpuUsageStr = node.CPUUsage.Value
			memUsageStr = node.MemoryUsage.Value
		}
	}

	avgCPUUsage := totalCPUUsage / float64(len(nodeMetrics))
	avgMemUsage := totalMemUsage / float64(len(nodeMetrics))

	return &ResourceUsage{
		UsedCPU: ResourceValue{
			Value: cpuUsageStr,
			Usage: avgCPUUsage,
		},
		UsedMemory: ResourceValue{
			Value: memUsageStr,
			Usage: avgMemUsage,
		},
		CPUPercentage:    avgCPUUsage,
		MemoryPercentage: avgMemUsage,
	}, nil
}

func (a *AWSMonitor) calculateOverallHealth(status *HealthStatus) ClusterHealthStatus {
	if len(status.Errors) > 0 {
		return HealthStatusUnhealthy
	}

	if len(status.Warnings) > 0 {
		return HealthStatusWarning
	}

	unhealthyNodes := 0
	for _, node := range status.Nodes {
		if node.Status != NodeHealthy {
			unhealthyNodes++
		}
	}

	if unhealthyNodes > 0 {
		if unhealthyNodes == len(status.Nodes) {
			return HealthStatusUnhealthy
		}
		return HealthStatusWarning
	}

	if status.Pods != nil && status.Pods.FailedPods > 0 {
		failureRate := float64(status.Pods.FailedPods) / float64(status.Pods.TotalPods)
		if failureRate > 0.1 {
			return HealthStatusUnhealthy
		} else if failureRate > 0.05 {
			return HealthStatusWarning
		}
	}

	return HealthStatusHealthy
}

func (a *AWSMonitor) updateKubeConfig(ctx context.Context, clusterName string) error {
	cmd := exec.CommandContext(ctx, "aws", "eks", "update-kubeconfig",
		"--region", a.region,
		"--name", clusterName)

	if a.profile != "" {
		cmd.Args = append(cmd.Args, "--profile", a.profile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update kubeconfig: %s", string(output))
	}

	return nil
}

func (a *AWSMonitor) getAccountID() string {
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