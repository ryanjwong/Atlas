package providers

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestLocalProvider_ValidateConfig(t *testing.T) {
	provider := &LocalProvider{}

	tests := []struct {
		name        string
		config      *ClusterConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid basic config",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 2,
			},
			wantErr: false,
		},
		{
			name: "empty cluster name",
			config: &ClusterConfig{
				Name:      "",
				NodeCount: 1,
			},
			wantErr:     true,
			errContains: "cluster name is required",
		},
		{
			name: "cluster name with spaces",
			config: &ClusterConfig{
				Name:      "test cluster",
				NodeCount: 1,
			},
			wantErr:     true,
			errContains: "cannot contain spaces",
		},
		{
			name: "negative node count",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: -1,
			},
			wantErr:     true,
			errContains: "cannot be negative",
		},
		{
			name: "too many nodes",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 15,
			},
			wantErr:     true,
			errContains: "cannot exceed 10",
		},
		{
			name: "valid network config",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				NetworkConfig: &NetworkConfig{
					PodCIDR:       "10.244.0.0/16",
					ServiceCIDR:   "10.96.0.0/12",
					APIServerPort: 8443,
					NetworkPlugin: "flannel",
					Ingress: &IngressConfig{
						Enabled:    true,
						Controller: "nginx",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid API server port",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				NetworkConfig: &NetworkConfig{
					APIServerPort: 500,
				},
			},
			wantErr:     true,
			errContains: "must be between 1024 and 65535",
		},
		{
			name: "invalid network plugin",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				NetworkConfig: &NetworkConfig{
					NetworkPlugin: "invalid-plugin",
				},
			},
			wantErr:     true,
			errContains: "invalid network plugin",
		},
		{
			name: "invalid ingress controller",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				NetworkConfig: &NetworkConfig{
					Ingress: &IngressConfig{
						Enabled:    true,
						Controller: "invalid-controller",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid ingress controller",
		},
		{
			name: "valid security config",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				SecurityConfig: &SecurityConfig{
					RBAC: &RBACConfig{
						Enabled: true,
					},
					AuthenticationMode: "RBAC",
					AuditLogging: &AuditConfig{
						Enabled:  true,
						LogLevel: "5",
					},
					ImageSecurity: &ImageSecurityConfig{
						ScanEnabled:            true,
						VulnerabilityThreshold: "medium",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid authentication mode",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				SecurityConfig: &SecurityConfig{
					AuthenticationMode: "invalid-mode",
				},
			},
			wantErr:     true,
			errContains: "invalid authentication mode",
		},
		{
			name: "invalid audit log level",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				SecurityConfig: &SecurityConfig{
					AuditLogging: &AuditConfig{
						Enabled:  true,
						LogLevel: "invalid",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid audit log level",
		},
		{
			name: "invalid vulnerability threshold",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				SecurityConfig: &SecurityConfig{
					ImageSecurity: &ImageSecurityConfig{
						VulnerabilityThreshold: "invalid",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid vulnerability threshold",
		},
		{
			name: "valid resource config",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				ResourceConfig: &ResourceConfig{
					AutoScaling: &AutoScalingConfig{
						Enabled:   true,
						MinNodes:  1,
						MaxNodes:  5,
						TargetCPU: 70,
					},
					Storage: &StorageConfig{
						StorageClasses: []StorageClassConfig{
							{
								Name:        "fast",
								Provisioner: "hostpath",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid autoscaling min nodes",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				ResourceConfig: &ResourceConfig{
					AutoScaling: &AutoScalingConfig{
						MinNodes: 0,
						MaxNodes: 5,
					},
				},
			},
			wantErr:     true,
			errContains: "minimum nodes must be at least 1",
		},
		{
			name: "invalid autoscaling max nodes",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				ResourceConfig: &ResourceConfig{
					AutoScaling: &AutoScalingConfig{
						MinNodes: 1,
						MaxNodes: 15,
					},
				},
			},
			wantErr:     true,
			errContains: "maximum nodes cannot exceed 10",
		},
		{
			name: "invalid autoscaling target CPU",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				ResourceConfig: &ResourceConfig{
					AutoScaling: &AutoScalingConfig{
						MinNodes:  1,
						MaxNodes:  5,
						TargetCPU: 95,
					},
				},
			},
			wantErr:     true,
			errContains: "target CPU must be between 10 and 90",
		},
		{
			name: "invalid storage provisioner",
			config: &ClusterConfig{
				Name:      "test-cluster",
				NodeCount: 1,
				ResourceConfig: &ResourceConfig{
					Storage: &StorageConfig{
						StorageClasses: []StorageClassConfig{
							{
								Name:        "test",
								Provisioner: "invalid-provisioner",
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid storage provisioner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require minikube if not available
			if !isMinikubeAvailable() {
				t.Skip("minikube not available")
			}

			err := provider.ValidateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateConfig() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfig() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestLocalProvider_GetProviderName(t *testing.T) {
	provider := &LocalProvider{}
	if got := provider.GetProviderName(); got != "local" {
		t.Errorf("GetProviderName() = %v, want %v", got, "local")
	}
}

func TestLocalProvider_GetSupportedRegions(t *testing.T) {
	provider := &LocalProvider{}
	regions := provider.GetSupportedRegions()
	if len(regions) == 0 {
		t.Error("GetSupportedRegions() should return at least one region")
	}
	if regions[0] != "local" {
		t.Errorf("GetSupportedRegions()[0] = %v, want %v", regions[0], "local")
	}
}

func TestLocalProvider_GetSupportedVersions(t *testing.T) {
	provider := &LocalProvider{}
	versions := provider.GetSupportedVersions()
	if len(versions) == 0 {
		t.Error("GetSupportedVersions() should return at least one version")
	}
}

func TestNetworkConfigValidation(t *testing.T) {
	provider := &LocalProvider{}

	tests := []struct {
		name        string
		netConfig   *NetworkConfig
		wantErr     bool
		errContains string
	}{
		{
			name:      "nil config",
			netConfig: nil,
			wantErr:   false,
		},
		{
			name: "valid port mapping",
			netConfig: &NetworkConfig{
				ExtraPortMaps: []PortMapping{
					{
						HostPort:      8080,
						ContainerPort: 80,
						Protocol:      "tcp",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port mapping - zero host port",
			netConfig: &NetworkConfig{
				ExtraPortMaps: []PortMapping{
					{
						HostPort:      0,
						ContainerPort: 80,
					},
				},
			},
			wantErr:     true,
			errContains: "positive port numbers",
		},
		{
			name: "invalid protocol",
			netConfig: &NetworkConfig{
				ExtraPortMaps: []PortMapping{
					{
						HostPort:      8080,
						ContainerPort: 80,
						Protocol:      "invalid",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateNetworkConfig(tt.netConfig)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateNetworkConfig() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validateNetworkConfig() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateNetworkConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

// Integration tests that require minikube
func TestLocalProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	if !isMinikubeAvailable() {
		t.Skip("minikube not available")
	}

	provider := &LocalProvider{}
	ctx := context.Background()
	testCluster := "atlas-test-integration"

	// Ensure cleanup
	defer func() {
		provider.DeleteCluster(ctx, testCluster)
	}()

	t.Run("CreateAndGetCluster", func(t *testing.T) {
		config := &ClusterConfig{
			Name:      testCluster,
			NodeCount: 1,
			Version:   "v1.31.0",
		}

		// Create cluster
		cluster, err := provider.CreateCluster(ctx, config)
		if err != nil {
			t.Fatalf("CreateCluster() error = %v", err)
		}

		if cluster.Name != testCluster {
			t.Errorf("CreateCluster() cluster name = %v, want %v", cluster.Name, testCluster)
		}

		// Get cluster
		retrievedCluster, err := provider.GetCluster(ctx, testCluster)
		if err != nil {
			t.Fatalf("GetCluster() error = %v", err)
		}

		if retrievedCluster.Name != testCluster {
			t.Errorf("GetCluster() cluster name = %v, want %v", retrievedCluster.Name, testCluster)
		}
	})

	t.Run("ListClusters", func(t *testing.T) {
		clusters, err := provider.ListClusters(ctx)
		if err != nil {
			t.Fatalf("ListClusters() error = %v", err)
		}

		found := false
		for _, cluster := range clusters {
			if cluster.Name == testCluster {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("ListClusters() should include test cluster %v", testCluster)
		}
	})

	t.Run("StopAndStartCluster", func(t *testing.T) {
		// Stop cluster
		err := provider.StopCluster(ctx, testCluster)
		if err != nil {
			t.Fatalf("StopCluster() error = %v", err)
		}

		// Check status
		cluster, err := provider.GetCluster(ctx, testCluster)
		if err != nil {
			t.Fatalf("GetCluster() after stop error = %v", err)
		}

		if cluster.Status != ClusterStatusStopped {
			t.Errorf("GetCluster() after stop status = %v, want %v", cluster.Status, ClusterStatusStopped)
		}

		// Start cluster
		err = provider.StartCluster(ctx, testCluster)
		if err != nil {
			t.Fatalf("StartCluster() error = %v", err)
		}
	})
}

// Benchmark tests
func BenchmarkLocalProvider_ValidateConfig(b *testing.B) {
	provider := &LocalProvider{}
	config := &ClusterConfig{
		Name:      "benchmark-cluster",
		NodeCount: 2,
		NetworkConfig: &NetworkConfig{
			PodCIDR:     "10.244.0.0/16",
			ServiceCIDR: "10.96.0.0/12",
		},
		SecurityConfig: &SecurityConfig{
			RBAC: &RBACConfig{Enabled: true},
		},
		ResourceConfig: &ResourceConfig{
			Limits: &ResourceLimits{
				CPU:    "2",
				Memory: "4Gi",
			},
		},
	}

	if !isMinikubeAvailable() {
		b.Skip("minikube not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.ValidateConfig(config)
	}
}

// Helper functions
func isMinikubeAvailable() bool {
	if os.Getenv("CI") == "true" {
		return false // Skip in CI environments
	}
	
	// Try to run minikube version
	cmd := exec.Command("minikube", "version")
	err := cmd.Run()
	return err == nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}