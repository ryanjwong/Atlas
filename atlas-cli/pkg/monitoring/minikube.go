package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type MinikubeMonitor struct {
	activeMonitoring map[string]context.CancelFunc
}

func NewMinikubeMonitor() *MinikubeMonitor {
	return &MinikubeMonitor{
		activeMonitoring: make(map[string]context.CancelFunc),
	}
}

func (m *MinikubeMonitor) GetMonitorName() string {
	return "minikube"
}

func (m *MinikubeMonitor) CheckClusterHealth(ctx context.Context, clusterName string) (*HealthStatus, error) {
	startTime := time.Now()
	
	status := &HealthStatus{
		ClusterName:   clusterName,
		OverallStatus: HealthStatusUnknown,
		LastChecked:   startTime,
		Warnings:      []string{},
		Errors:        []string{},
	}
	
	if !m.isMinikubeRunning(ctx, clusterName) {
		status.OverallStatus = HealthStatusUnhealthy
		status.Errors = append(status.Errors, "Minikube cluster is not running")
		status.CheckDuration = time.Since(startTime)
		return status, nil
	}
	
	controlPlaneHealth, err := m.checkControlPlane(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Control plane check failed: %v", err))
	} else {
		status.ControlPlane = controlPlaneHealth
	}
	
	nodes, err := m.checkNodes(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Node check failed: %v", err))
	} else {
		status.Nodes = nodes
	}
	
	podHealth, err := m.checkPods(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Pod check failed: %v", err))
	} else {
		status.Pods = podHealth
	}
	
	serviceHealth, err := m.checkServices(ctx, clusterName)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Service check failed: %v", err))
	} else {
		status.Services = serviceHealth
	}
	
	status.OverallStatus = m.calculateOverallHealth(status)
	status.CheckDuration = time.Since(startTime)
	
	return status, nil
}

func (m *MinikubeMonitor) GetClusterMetrics(ctx context.Context, clusterName string) (*ClusterMetrics, error) {
	timestamp := time.Now()
	
	metrics := &ClusterMetrics{
		ClusterName: clusterName,
		Timestamp:   timestamp,
	}
	
	if !m.isMinikubeRunning(ctx, clusterName) {
		return nil, fmt.Errorf("cluster %s is not running", clusterName)
	}
	
	nodeMetrics, err := m.getNodeMetrics(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}
	metrics.NodeMetrics = nodeMetrics
	
	podMetrics, err := m.getPodMetrics(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}
	metrics.PodMetrics = podMetrics
	
	resourceUsage, err := m.calculateResourceUsage(nodeMetrics, podMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate resource usage: %w", err)
	}
	metrics.ResourceUsage = resourceUsage
	
	return metrics, nil
}

func (m *MinikubeMonitor) StartMonitoring(ctx context.Context, config *MonitoringConfig) error {
	for _, clusterName := range config.ClusterNames {
		if _, exists := m.activeMonitoring[clusterName]; exists {
			continue
		}
		
		monitorCtx, cancel := context.WithCancel(ctx)
		m.activeMonitoring[clusterName] = cancel
		
		go m.monitorCluster(monitorCtx, clusterName, config)
	}
	
	return nil
}

func (m *MinikubeMonitor) StopMonitoring(ctx context.Context, clusterName string) error {
	if cancel, exists := m.activeMonitoring[clusterName]; exists {
		cancel()
		delete(m.activeMonitoring, clusterName)
	}
	
	return nil
}

func (m *MinikubeMonitor) monitorCluster(ctx context.Context, clusterName string, config *MonitoringConfig) {
	healthTicker := time.NewTicker(config.CheckInterval)
	metricsTicker := time.NewTicker(config.MetricsInterval)
	
	defer healthTicker.Stop()
	defer metricsTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-healthTicker.C:
			_, err := m.CheckClusterHealth(ctx, clusterName)
			if err != nil && config.EnableAlerts {
				fmt.Printf("Health check failed for cluster %s: %v\n", clusterName, err)
			}
		case <-metricsTicker.C:
			_, err := m.GetClusterMetrics(ctx, clusterName)
			if err != nil && config.EnableAlerts {
				fmt.Printf("Metrics collection failed for cluster %s: %v\n", clusterName, err)
			}
		}
	}
}

func (m *MinikubeMonitor) isMinikubeRunning(ctx context.Context, clusterName string) bool {
	cmd := exec.CommandContext(ctx, "minikube", "status", "-p", clusterName, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err != nil {
		return false
	}
	
	if host, ok := statusData["Host"].(string); ok && strings.ToLower(host) == "running" {
		if kubelet, ok := statusData["Kubelet"].(string); ok && strings.ToLower(kubelet) == "running" {
			return true
		}
	}
	
	return false
}

func (m *MinikubeMonitor) checkControlPlane(ctx context.Context, clusterName string) (*ControlPlaneHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "componentstatuses", "-o", "json", "--context", clusterName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get component status: %w", err)
	}
	
	var components struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
				Message string `json:"message,omitempty"`
			} `json:"conditions"`
		} `json:"items"`
	}
	
	if err := json.Unmarshal(output, &components); err != nil {
		return nil, fmt.Errorf("failed to parse component status: %w", err)
	}
	
	health := &ControlPlaneHealth{
		APIServer:         ComponentStatus{Status: ComponentUnknown, LastCheck: time.Now()},
		Scheduler:         ComponentStatus{Status: ComponentUnknown, LastCheck: time.Now()},
		ControllerManager: ComponentStatus{Status: ComponentUnknown, LastCheck: time.Now()},
		Etcd:              ComponentStatus{Status: ComponentUnknown, LastCheck: time.Now()},
	}
	
	for _, component := range components.Items {
		name := component.Metadata.Name
		status := ComponentUnknown
		message := ""
		
		for _, condition := range component.Conditions {
			if condition.Type == "Healthy" {
				if condition.Status == "True" {
					status = ComponentHealthy
				} else {
					status = ComponentUnhealthy
					message = condition.Message
				}
				break
			}
		}
		
		componentStatus := ComponentStatus{
			Status:    status,
			Message:   message,
			LastCheck: time.Now(),
		}
		
		switch name {
		case "scheduler":
			health.Scheduler = componentStatus
		case "controller-manager":
			health.ControllerManager = componentStatus
		case "etcd-0":
			health.Etcd = componentStatus
		}
	}
	
	health.APIServer = ComponentStatus{
		Status:    ComponentHealthy,
		LastCheck: time.Now(),
	}
	
	return health, nil
}

func (m *MinikubeMonitor) checkNodes(ctx context.Context, clusterName string) ([]NodeHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "json", "--context", clusterName)
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

func (m *MinikubeMonitor) checkPods(ctx context.Context, clusterName string) (*PodHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "--all-namespaces", "-o", "json", "--context", clusterName)
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

func (m *MinikubeMonitor) checkServices(ctx context.Context, clusterName string) (*ServiceHealth, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "services", "--all-namespaces", "-o", "json", "--context", clusterName)
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

func (m *MinikubeMonitor) getNodeMetrics(ctx context.Context, clusterName string) ([]NodeMetrics, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "nodes", "--context", clusterName, "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
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
		cpuPercent := strings.TrimSuffix(fields[2], "%")
		memUsage := fields[3]
		memPercent := strings.TrimSuffix(fields[4], "%")
		
		cpuPercentFloat, _ := strconv.ParseFloat(cpuPercent, 64)
		memPercentFloat, _ := strconv.ParseFloat(memPercent, 64)
		
		metrics = append(metrics, NodeMetrics{
			NodeName: nodeName,
			CPUUsage: ResourceValue{
				Value: cpuUsage,
				Usage: cpuPercentFloat,
			},
			MemoryUsage: ResourceValue{
				Value: memUsage,
				Usage: memPercentFloat,
			},
			Timestamp: time.Now(),
		})
	}
	
	return metrics, nil
}

func (m *MinikubeMonitor) getPodMetrics(ctx context.Context, clusterName string) ([]PodMetrics, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "top", "pods", "--all-namespaces", "--context", clusterName, "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
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

func (m *MinikubeMonitor) calculateResourceUsage(nodeMetrics []NodeMetrics, podMetrics []PodMetrics) (*ResourceUsage, error) {
	if len(nodeMetrics) == 0 {
		return nil, fmt.Errorf("no node metrics available")
	}
	
	var totalCPUUsage, totalMemUsage float64
	var cpuUsageStr, memUsageStr string
	
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

func (m *MinikubeMonitor) calculateOverallHealth(status *HealthStatus) ClusterHealthStatus {
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