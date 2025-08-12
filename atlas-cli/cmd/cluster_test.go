package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestLoadClusterConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		errContains string
		checkFunc   func(*providers.ClusterConfig) bool
	}{
		{
			name: "valid basic config",
			configYAML: `
name: test-cluster
region: local
nodeCount: 2
version: v1.31.0
`,
			wantErr: false,
			checkFunc: func(c *providers.ClusterConfig) bool {
				return c.Name == "test-cluster" && c.NodeCount == 2 && c.Version == "v1.31.0"
			},
		},
		{
			name: "valid config with network settings",
			configYAML: `
name: test-cluster
region: local
nodeCount: 1
networkConfig:
  podCIDR: 10.244.0.0/16
  serviceCIDR: 10.96.0.0/12
  ingress:
    enabled: true
    controller: nginx
`,
			wantErr: false,
			checkFunc: func(c *providers.ClusterConfig) bool {
				return c.NetworkConfig != nil &&
					c.NetworkConfig.PodCIDR == "10.244.0.0/16" &&
					c.NetworkConfig.Ingress != nil &&
					c.NetworkConfig.Ingress.Enabled
			},
		},
		{
			name: "valid config with security settings",
			configYAML: `
name: test-cluster
region: local
nodeCount: 1
securityConfig:
  rbac:
    enabled: true
  networkPolicy:
    enabled: true
    defaultPolicy: deny-all
  auditLogging:
    enabled: true
    logLevel: "3"
`,
			wantErr: false,
			checkFunc: func(c *providers.ClusterConfig) bool {
				return c.SecurityConfig != nil &&
					c.SecurityConfig.RBAC != nil &&
					c.SecurityConfig.RBAC.Enabled &&
					c.SecurityConfig.NetworkPolicy != nil &&
					c.SecurityConfig.NetworkPolicy.Enabled
			},
		},
		{
			name: "valid config with resource settings",
			configYAML: `
name: test-cluster
region: local
nodeCount: 1
resourceConfig:
  limits:
    cpu: "4"
    memory: 8Gi
  autoScaling:
    enabled: true
    minNodes: 1
    maxNodes: 5
  monitoring:
    enabled: true
    prometheus:
      enabled: true
`,
			wantErr: false,
			checkFunc: func(c *providers.ClusterConfig) bool {
				return c.ResourceConfig != nil &&
					c.ResourceConfig.Limits != nil &&
					c.ResourceConfig.Limits.CPU == "4" &&
					c.ResourceConfig.AutoScaling != nil &&
					c.ResourceConfig.AutoScaling.Enabled
			},
		},
		{
			name:        "invalid YAML",
			configYAML:  `invalid: yaml: content:`,
			wantErr:     true,
			errContains: "parse YAML",
		},
		{
			name:        "empty file",
			configYAML:  "",
			wantErr:     false,
			checkFunc:   func(c *providers.ClusterConfig) bool { return c.Name == "" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Load config
			config, err := loadClusterConfig(configFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadClusterConfig() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("loadClusterConfig() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadClusterConfig() unexpected error = %v", err)
				return
			}

			if tt.checkFunc != nil && !tt.checkFunc(config) {
				t.Errorf("loadClusterConfig() config validation failed")
			}
		})
	}
}

func TestLoadClusterConfig_FileNotFound(t *testing.T) {
	_, err := loadClusterConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("loadClusterConfig() should fail for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("loadClusterConfig() error should mention file read failure, got: %v", err)
	}
}

func TestClusterGenerateConfigCmd(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		flags        map[string]string
		wantErr      bool
		checkFile    bool
	}{
		{
			name:    "generate config to stdout",
			args:    []string{"test-cluster"},
			wantErr: false,
		},
		{
			name:      "generate config to file",
			args:      []string{"test-cluster"},
			flags:     map[string]string{"output": "test-config.yaml"},
			wantErr:   false,
			checkFile: true,
		},
		{
			name:    "missing cluster name",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for output files
			tmpDir := t.TempDir()
			originalWd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(originalWd)

			// Create a standalone command for testing
			cmd := &cobra.Command{
				Use:   "generate-config [name]",
				Short: "Generate a sample configuration file",
				Args:  cobra.ExactArgs(1),
				RunE:  clusterGenerateConfigCmd.RunE,
			}
			
			// Add flags
			cmd.Flags().StringP("output", "o", "", "Output file path")

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}

			// Set args and run
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("clusterGenerateConfigCmd expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("clusterGenerateConfigCmd unexpected error = %v", err)
				return
			}

			// We don't check stdout output as it may go directly to os.Stdout
			// The important thing is that the command succeeds

			// Check file creation
			if tt.checkFile {
				outputFile := tt.flags["output"]
				if _, err := os.Stat(outputFile); os.IsNotExist(err) {
					t.Errorf("clusterGenerateConfigCmd should have created file %s", outputFile)
				} else {
					// Validate the generated file is valid YAML
					data, err := os.ReadFile(outputFile)
					if err != nil {
						t.Errorf("failed to read generated config file: %v", err)
					}
					var config providers.ClusterConfig
					if err := yaml.Unmarshal(data, &config); err != nil {
						t.Errorf("generated config is not valid YAML: %v", err)
					}
				}
			}

			// Reset flags
			for flag := range tt.flags {
				cmd.Flags().Set(flag, "")
			}
		})
	}
}

func TestClusterCreateCmd_FlagParsing(t *testing.T) {
	// We can't mock GetServices easily, so we'll test the flag parsing logic directly

	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		wantErr     bool
		errContains string
		checkConfig func(*providers.ClusterConfig) bool
	}{
		{
			name: "basic flags",
			args: []string{"test-cluster"},
			flags: map[string]string{
				"nodes":    "3",
				"version":  "v1.30.0",
				"provider": "local",
			},
			wantErr: false,
			checkConfig: func(c *providers.ClusterConfig) bool {
				return c.Name == "test-cluster" && c.NodeCount == 3 && c.Version == "v1.30.0"
			},
		},
		{
			name: "networking flags",
			args: []string{"test-cluster"},
			flags: map[string]string{
				"enable-ingress":      "true",
				"enable-load-balancer": "true",
				"api-server-port":     "9443",
			},
			wantErr: false,
			checkConfig: func(c *providers.ClusterConfig) bool {
				return c.NetworkConfig != nil &&
					c.NetworkConfig.Ingress != nil &&
					c.NetworkConfig.Ingress.Enabled &&
					c.NetworkConfig.LoadBalancer != nil &&
					c.NetworkConfig.LoadBalancer.Enabled &&
					c.NetworkConfig.APIServerPort == 9443
			},
		},
		{
			name: "security flags",
			args: []string{"test-cluster"},
			flags: map[string]string{
				"enable-rbac":           "true",
				"enable-network-policy": "true",
			},
			wantErr: false,
			checkConfig: func(c *providers.ClusterConfig) bool {
				return c.SecurityConfig != nil &&
					c.SecurityConfig.RBAC != nil &&
					c.SecurityConfig.RBAC.Enabled &&
					c.SecurityConfig.NetworkPolicy != nil &&
					c.SecurityConfig.NetworkPolicy.Enabled
			},
		},
		{
			name: "resource flags",
			args: []string{"test-cluster"},
			flags: map[string]string{
				"enable-monitoring": "true",
				"cpu-limit":         "4",
				"memory-limit":      "8Gi",
			},
			wantErr: false,
			checkConfig: func(c *providers.ClusterConfig) bool {
				return c.ResourceConfig != nil &&
					c.ResourceConfig.Monitoring != nil &&
					c.ResourceConfig.Monitoring.Enabled &&
					c.ResourceConfig.Limits != nil &&
					c.ResourceConfig.Limits.CPU == "4" &&
					c.ResourceConfig.Limits.Memory == "8Gi"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command to test flag parsing logic
			cmd := &cobra.Command{
				Use: "create",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Test flag parsing logic directly
					clusterName := args[0]
					config := buildConfigFromFlags(cmd, clusterName)

					// Run the test check
					if tt.checkConfig != nil && !tt.checkConfig(config) {
						t.Errorf("Config validation failed for test %s", tt.name)
					}

					return nil
				},
			}

			// Add the same flags as clusterCreateCmd
			cmd.Flags().StringP("provider", "p", "local", "Cloud provider")
			cmd.Flags().StringP("region", "r", "local", "Region")
			cmd.Flags().IntP("nodes", "n", 1, "Number of nodes")
			cmd.Flags().StringP("version", "k", "", "Kubernetes version")
			cmd.Flags().String("instance-type", "", "Instance type")
			cmd.Flags().StringP("config", "c", "", "Config file path")
			cmd.Flags().Bool("enable-ingress", false, "Enable ingress")
			cmd.Flags().Bool("enable-load-balancer", false, "Enable load balancer")
			cmd.Flags().Bool("enable-rbac", false, "Enable RBAC")
			cmd.Flags().Bool("enable-network-policy", false, "Enable network policies")
			cmd.Flags().Bool("enable-monitoring", false, "Enable monitoring")
			cmd.Flags().Int("api-server-port", 0, "API server port")
			cmd.Flags().String("cpu-limit", "", "CPU limit")
			cmd.Flags().String("memory-limit", "", "Memory limit")

			// Set flags
			for flag, value := range tt.flags {
				if err := cmd.Flags().Set(flag, value); err != nil {
					t.Fatalf("failed to set flag %s=%s: %v", flag, value, err)
				}
			}

			// Set args and run
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error = %v", err)
			}
		})
	}
}

// Helper function to build config from flags (extracted from clusterCreateCmd logic)
func buildConfigFromFlags(cmd *cobra.Command, clusterName string) *providers.ClusterConfig {
	configFile, _ := cmd.Flags().GetString("config")
	var config *providers.ClusterConfig

	if configFile != "" {
		var err error
		config, err = loadClusterConfig(configFile)
		if err != nil {
			return nil
		}
		config.Name = clusterName
	} else {
		region, _ := cmd.Flags().GetString("region")
		nodeCount, _ := cmd.Flags().GetInt("nodes")
		version, _ := cmd.Flags().GetString("version")
		instanceType, _ := cmd.Flags().GetString("instance-type")

		config = &providers.ClusterConfig{
			Name:         clusterName,
			Region:       region,
			NodeCount:    nodeCount,
			Version:      version,
			InstanceType: instanceType,
		}

		enableIngress, _ := cmd.Flags().GetBool("enable-ingress")
		enableLoadBalancer, _ := cmd.Flags().GetBool("enable-load-balancer")
		enableRBAC, _ := cmd.Flags().GetBool("enable-rbac")
		enableNetworkPolicy, _ := cmd.Flags().GetBool("enable-network-policy")
		enableMonitoring, _ := cmd.Flags().GetBool("enable-monitoring")
		apiServerPort, _ := cmd.Flags().GetInt("api-server-port")
		cpuLimit, _ := cmd.Flags().GetString("cpu-limit")
		memoryLimit, _ := cmd.Flags().GetString("memory-limit")

		if enableIngress || enableLoadBalancer || apiServerPort > 0 {
			config.NetworkConfig = &providers.NetworkConfig{}
			if enableIngress {
				config.NetworkConfig.Ingress = &providers.IngressConfig{Enabled: true}
			}
			if enableLoadBalancer {
				config.NetworkConfig.LoadBalancer = &providers.LoadBalancerConfig{Enabled: true}
			}
			if apiServerPort > 0 {
				config.NetworkConfig.APIServerPort = apiServerPort
			}
		}

		if enableRBAC || enableNetworkPolicy {
			config.SecurityConfig = &providers.SecurityConfig{}
			if enableRBAC {
				config.SecurityConfig.RBAC = &providers.RBACConfig{Enabled: true}
			}
			if enableNetworkPolicy {
				config.SecurityConfig.NetworkPolicy = &providers.NetworkPolicyConfig{Enabled: true}
			}
		}

		if enableMonitoring || cpuLimit != "" || memoryLimit != "" {
			config.ResourceConfig = &providers.ResourceConfig{}
			if enableMonitoring {
				config.ResourceConfig.Monitoring = &providers.MonitoringConfig{
					Enabled: true,
					Prometheus: &providers.PrometheusConfig{Enabled: true},
				}
			}
			if cpuLimit != "" || memoryLimit != "" {
				config.ResourceConfig.Limits = &providers.ResourceLimits{
					CPU:    cpuLimit,
					Memory: memoryLimit,
				}
			}
		}
	}

	return config
}

func TestConfigFileVsFlagsIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a test config file
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: file-cluster
region: local
nodeCount: 3
version: v1.30.0
networkConfig:
  ingress:
    enabled: true
securityConfig:
  rbac:
    enabled: true
resourceConfig:
  limits:
    cpu: "2"
    memory: 4Gi
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	// Test loading the config
	config, err := loadClusterConfig(configFile)
	if err != nil {
		t.Fatalf("loadClusterConfig() error = %v", err)
	}

	// Validate the loaded config
	if config.Name != "file-cluster" {
		t.Errorf("config.Name = %v, want file-cluster", config.Name)
	}
	if config.NodeCount != 3 {
		t.Errorf("config.NodeCount = %v, want 3", config.NodeCount)
	}
	if config.NetworkConfig == nil || config.NetworkConfig.Ingress == nil || !config.NetworkConfig.Ingress.Enabled {
		t.Error("Ingress should be enabled from config file")
	}
	if config.SecurityConfig == nil || config.SecurityConfig.RBAC == nil || !config.SecurityConfig.RBAC.Enabled {
		t.Error("RBAC should be enabled from config file")
	}
	if config.ResourceConfig == nil || config.ResourceConfig.Limits == nil {
		t.Error("Resource limits should be set from config file")
	}
}