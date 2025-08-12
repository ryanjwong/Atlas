package logsource

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MinikubeLogSource implements LogSource using minikube's audit logs and commands
type MinikubeLogSource struct{}

// NewMinikubeLogSource creates a new minikube log source
func NewMinikubeLogSource() *MinikubeLogSource {
	return &MinikubeLogSource{}
}

// MinikubeAuditEntry represents a raw entry from minikube audit logs
type MinikubeAuditEntry struct {
	Command   string
	Args      string
	Profile   string
	User      string
	Version   string
	StartTime time.Time
	EndTime   *time.Time
	Duration  *time.Duration
}

// MinikubeProfilesResponse represents minikube profile list response
type MinikubeProfilesResponse struct {
	Invalid []interface{} `json:"invalid"`
	Valid   []Profile     `json:"valid"`
}

// Profile represents a minikube profile
type Profile struct {
	Name string `json:"Name"`
}

func (m *MinikubeLogSource) GetSourceName() string {
	return "minikube"
}

func (m *MinikubeLogSource) GetClusterHistory(ctx context.Context, clusterName string, limit int) ([]*OperationHistory, error) {
	cmd := exec.CommandContext(ctx, "minikube", "logs", "--audit", "-n", strconv.Itoa(limit*2))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get minikube audit logs: %w", err)
	}

	entries := parseMinikubeAudit(string(output))
	
	var history []*OperationHistory
	for _, entry := range entries {
		if entry.Profile == clusterName {
			opHistory := convertToOperationHistory(entry)
			if opHistory != nil {
				history = append(history, opHistory)
			}
		}
		
		if len(history) >= limit {
			break
		}
	}

	return history, nil
}

func (m *MinikubeLogSource) GetAllClustersHistory(ctx context.Context, limit int) (map[string][]*OperationHistory, error) {
	cmd := exec.CommandContext(ctx, "minikube", "logs", "--audit", "-n", strconv.Itoa(limit*5))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get minikube audit logs: %w", err)
	}

	entries := parseMinikubeAudit(string(output))
	
	clusterHistories := make(map[string][]*OperationHistory)
	
	for _, entry := range entries {
		if entry.Profile != "" {
			opHistory := convertToOperationHistory(entry)
			if opHistory != nil {
				clusterHistories[entry.Profile] = append(clusterHistories[entry.Profile], opHistory)
			}
		}
	}

	return clusterHistories, nil
}

func (m *MinikubeLogSource) ListClusters(ctx context.Context) ([]*ClusterInfo, error) {
	cmd := exec.CommandContext(ctx, "minikube", "profile", "list", "-o=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list minikube profiles: %w", err)
	}

	var profiles MinikubeProfilesResponse
	if err := json.Unmarshal(output, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse minikube profiles: %w", err)
	}

	var clusters []*ClusterInfo
	for _, profile := range profiles.Valid {
		clusterStatus, err := m.GetClusterStatus(ctx, profile.Name)
		if err != nil {
			continue
		}

		history, _ := m.GetClusterHistory(ctx, profile.Name, 1000)
		var createdAt, updatedAt time.Time
		if len(history) > 0 {
			for i := len(history) - 1; i >= 0; i-- {
				if history[i].OperationType == OpTypeCreate {
					createdAt = history[i].StartedAt
					break
				}
			}
			updatedAt = history[0].StartedAt
		}

		if createdAt.IsZero() {
			createdAt = time.Now() 
		}
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		clusters = append(clusters, &ClusterInfo{
			Name:         profile.Name,
			Provider:     "local",
			Region:       "local",
			Status:       clusterStatus.Status,
			NodeCount:    clusterStatus.NodeCount,
			Version:      clusterStatus.Version,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
			Tags:         make(map[string]string),
			LastActivity: &updatedAt,
		})
	}

	return clusters, nil
}

func (m *MinikubeLogSource) GetClusterStatus(ctx context.Context, clusterName string) (*ClusterStatus, error) {
	cmd := exec.CommandContext(ctx, "minikube", "status", "-p", clusterName)
	output, err := cmd.CombinedOutput()
	statusStr := string(output)

	var status StatusType
	if strings.Contains(statusStr, "Running") {
		status = StatusRunning
	} else if strings.Contains(statusStr, "Stopped") {
		status = StatusStopped
	} else if err != nil {
		if strings.Contains(err.Error(), "exit status 7") || strings.Contains(statusStr, "does not exist") {
			return nil, fmt.Errorf("cluster %s does not exist", clusterName)
		}
		status = StatusError
	} else {
		status = StatusError
	}

	cmd = exec.CommandContext(ctx, "minikube", "ip", "-p", clusterName)
	ipOutput, err := cmd.CombinedOutput()
	var endpoint string
	if err == nil {
		endpoint = strings.TrimSpace(string(ipOutput))
	}

	var version string
	var nodeCount int = 1

	cmd = exec.CommandContext(ctx, "minikube", "profile", "list")
	profileOutput, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(profileOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, clusterName) {
				fields := strings.Fields(line)
				if len(fields) >= 8 {
					version = fields[5]
					if nodeCountStr := fields[7]; nodeCountStr != "" {
						if count, parseErr := strconv.Atoi(nodeCountStr); parseErr == nil {
							nodeCount = count
						}
					}
				}
				break
			}
		}
	}

	if version == "" && status == StatusRunning {
		cmd = exec.CommandContext(ctx, "minikube", "kubectl", "-p", clusterName, "--", "version", "--client=false", "--output=yaml")
		versionOutput, err := cmd.CombinedOutput()
		if err == nil {
			lines := strings.Split(string(versionOutput), "\n")
			for _, line := range lines {
				if strings.Contains(line, "gitVersion:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						version = strings.TrimSpace(strings.Trim(parts[1], " \""))
					}
					break
				}
			}
		}
	}

	if status == StatusRunning {
		cmd = exec.CommandContext(ctx, "minikube", "kubectl", "-p", clusterName, "--", "get", "nodes", "--no-headers")
		nodesOutput, err := cmd.CombinedOutput()
		if err == nil {
			nodeLines := strings.Split(strings.TrimSpace(string(nodesOutput)), "\n")
			if len(nodeLines) > 0 && nodeLines[0] != "" {
				nodeCount = len(nodeLines)
			}
		}
	}

	return &ClusterStatus{
		Name:      clusterName,
		Status:    status,
		NodeCount: nodeCount,
		Version:   version,
		Endpoint:  endpoint,
		Metadata: map[string]string{
			"source": "minikube",
		},
	}, nil
}

func parseMinikubeAudit(output string) []MinikubeAuditEntry {
	var entries []MinikubeAuditEntry
	
	lines := strings.Split(output, "\n")
	
	inTable := false
	for _, line := range lines {
		if strings.Contains(line, "| Command |") {
			inTable = true
			continue
		}
		
		if !inTable || !strings.HasPrefix(line, "|") {
			continue
		}
		
		if strings.Contains(line, "---") {
			continue
		}
		
		entry := parseAuditLine(line)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	
	return entries
}

func parseAuditLine(line string) *MinikubeAuditEntry {
	parts := strings.Split(line, "|")
	if len(parts) < 7 {
		return nil
	}
	
	command := strings.TrimSpace(parts[1])
	args := strings.TrimSpace(parts[2])
	profile := strings.TrimSpace(parts[3])
	user := strings.TrimSpace(parts[4])
	version := strings.TrimSpace(parts[5])
	startTimeStr := strings.TrimSpace(parts[6])
	endTimeStr := ""
	if len(parts) > 7 {
		endTimeStr = strings.TrimSpace(parts[7])
	}
	
	if command == "" || profile == "" {
		return nil
	}
	
	entry := &MinikubeAuditEntry{
		Command: command,
		Args:    args,
		Profile: profile,
		User:    user,
		Version: version,
	}
	
	if startTime, err := parseMinikubeTime(startTimeStr); err == nil {
		entry.StartTime = startTime
	}
	
	if endTimeStr != "" {
		if endTime, err := parseMinikubeTime(endTimeStr); err == nil {
			entry.EndTime = &endTime
			duration := endTime.Sub(entry.StartTime)
			entry.Duration = &duration
		}
	}
	
	return entry
}

func parseMinikubeTime(timeStr string) (time.Time, error) {
	formats := []string{
		"02 Jan 06 15:04 MST",
		"2 Jan 06 15:04 MST", 
		"02 Jan 06 15:04:05 MST",
		"2 Jan 06 15:04:05 MST",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			if t.Year() < 2000 {
				t = t.AddDate(2000, 0, 0)
			}
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

func convertToOperationHistory(entry MinikubeAuditEntry) *OperationHistory {
	var opType OperationType
	switch entry.Command {
	case "start":
		if strings.Contains(entry.Args, "--nodes") || strings.Contains(entry.Args, "--kubernetes-version") {
			opType = OpTypeCreate
		} else {
			opType = OpTypeStart
		}
	case "stop":
		opType = OpTypeStop
	case "delete":
		opType = OpTypeDelete
	default:
		return nil
	}
	
	status := OpStatusCompleted
	if entry.EndTime == nil {
		status = OpStatusRunning
	}
	
	details := make(map[string]interface{})
	if entry.Args != "" {
		details["args"] = entry.Args
		
		if nodes := extractNodeCount(entry.Args); nodes > 0 {
			details["nodeCount"] = nodes
		}
		if version := extractKubernetesVersion(entry.Args); version != "" {
			details["kubernetesVersion"] = version
		}
	}
	
	op := &OperationHistory{
		ClusterName:      entry.Profile,
		OperationType:    opType,
		OperationStatus:  status,
		StartedAt:        entry.StartTime,
		CompletedAt:      entry.EndTime,
		UserID:           entry.User,
		OperationDetails: details,
		Metadata: map[string]string{
			"minikube_version": entry.Version,
			"source":           "minikube_audit",
		},
	}
	
	if entry.Duration != nil {
		durationMS := float64(entry.Duration.Milliseconds())
		op.DurationMS = &durationMS
	}
	
	return op
}

func extractNodeCount(args string) int {
	re := regexp.MustCompile(`--nodes[=\s](\d+)`)
	matches := re.FindStringSubmatch(args)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count
		}
	}
	return 0
}

func extractKubernetesVersion(args string) string {
	re := regexp.MustCompile(`--kubernetes-version[=\s]([^\s]+)`)
	matches := re.FindStringSubmatch(args)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}