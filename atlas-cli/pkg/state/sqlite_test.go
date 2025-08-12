package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStateManager_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atlas-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	stateManager, err := NewSQLiteStateManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateManager.Disconnect(context.Background())

	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		err := stateManager.Health(ctx)
		if err != nil {
			t.Errorf("Health() error = %v", err)
		}
	})

	t.Run("Validate", func(t *testing.T) {
		err := stateManager.Validate(ctx)
		if err != nil {
			t.Errorf("Validate() error = %v", err)
		}
	})

	t.Run("Connect", func(t *testing.T) {
		err := stateManager.Connect(ctx)
		if err != nil {
			t.Errorf("Connect() error = %v", err)
		}
	})
}

func TestSQLiteStateManager_ClusterState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atlas-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	stateManager, err := NewSQLiteStateManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateManager.Disconnect(context.Background())

	ctx := context.Background()
	testCluster := &ClusterState{
		Name:      "test-cluster",
		Provider:  "local",
		Region:    "local",
		Status:    "creating",
		NodeCount: 2,
		Version:   "v1.31.0",
		Config: map[string]interface{}{
			"nodes": 2,
			"version": "v1.31.0",
		},
		Metadata: map[string]string{
			"creation_method": "test",
			"environment": "testing",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "test-user",
	}

	t.Run("SaveClusterState", func(t *testing.T) {
		err := stateManager.SaveClusterState(ctx, testCluster)
		if err != nil {
			t.Fatalf("SaveClusterState() error = %v", err)
		}
		
		if testCluster.ID == 0 {
			t.Error("SaveClusterState() should set cluster ID")
		}
	})

	t.Run("GetClusterState", func(t *testing.T) {
		retrievedCluster, err := stateManager.GetClusterState(ctx, testCluster.Name)
		if err != nil {
			t.Fatalf("GetClusterState() error = %v", err)
		}
		
		if retrievedCluster == nil {
			t.Fatal("GetClusterState() returned nil cluster")
		}
		
		if retrievedCluster.Name != testCluster.Name {
			t.Errorf("GetClusterState() name = %v, want %v", retrievedCluster.Name, testCluster.Name)
		}
		
		if retrievedCluster.Provider != testCluster.Provider {
			t.Errorf("GetClusterState() provider = %v, want %v", retrievedCluster.Provider, testCluster.Provider)
		}
		
		if retrievedCluster.Status != testCluster.Status {
			t.Errorf("GetClusterState() status = %v, want %v", retrievedCluster.Status, testCluster.Status)
		}
		
		if retrievedCluster.NodeCount != testCluster.NodeCount {
			t.Errorf("GetClusterState() nodeCount = %v, want %v", retrievedCluster.NodeCount, testCluster.NodeCount)
		}
		
		if retrievedCluster.CreatedBy != testCluster.CreatedBy {
			t.Errorf("GetClusterState() createdBy = %v, want %v", retrievedCluster.CreatedBy, testCluster.CreatedBy)
		}
	})

	t.Run("UpdateClusterState", func(t *testing.T) {
		testCluster.Status = "running"
		testCluster.UpdatedAt = time.Now()
		
		err := stateManager.SaveClusterState(ctx, testCluster)
		if err != nil {
			t.Fatalf("SaveClusterState() update error = %v", err)
		}
		
		retrievedCluster, err := stateManager.GetClusterState(ctx, testCluster.Name)
		if err != nil {
			t.Fatalf("GetClusterState() after update error = %v", err)
		}
		
		if retrievedCluster.Status != "running" {
			t.Errorf("GetClusterState() after update status = %v, want %v", retrievedCluster.Status, "running")
		}
	})

	t.Run("ListClusters", func(t *testing.T) {
		clusters, err := stateManager.ListClusters(ctx)
		if err != nil {
			t.Fatalf("ListClusters() error = %v", err)
		}
		
		if len(clusters) == 0 {
			t.Error("ListClusters() should return at least one cluster")
		}
		
		found := false
		for _, cluster := range clusters {
			if cluster.Name == testCluster.Name {
				found = true
				if cluster.Status != "running" {
					t.Errorf("ListClusters() cluster status = %v, want %v", cluster.Status, "running")
				}
				break
			}
		}
		
		if !found {
			t.Error("ListClusters() should include test cluster")
		}
	})

	t.Run("DeleteClusterState", func(t *testing.T) {
		err := stateManager.DeleteClusterState(ctx, testCluster.Name)
		if err != nil {
			t.Fatalf("DeleteClusterState() error = %v", err)
		}
		
		retrievedCluster, err := stateManager.GetClusterState(ctx, testCluster.Name)
		if err != nil {
			t.Fatalf("GetClusterState() after delete error = %v", err)
		}
		
		if retrievedCluster != nil {
			t.Error("GetClusterState() after delete should return nil")
		}
	})

	t.Run("GetNonExistentCluster", func(t *testing.T) {
		cluster, err := stateManager.GetClusterState(ctx, "non-existent")
		if err != nil {
			t.Fatalf("GetClusterState() for non-existent cluster error = %v", err)
		}
		
		if cluster != nil {
			t.Error("GetClusterState() for non-existent cluster should return nil")
		}
	})
}

func TestSQLiteStateManager_Resources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "atlas-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	stateManager, err := NewSQLiteStateManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateManager.Disconnect(context.Background())

	ctx := context.Background()
	clusterName := "test-cluster-resources"
	
	testCluster := &ClusterState{
		Name:      clusterName,
		Provider:  "local",
		Region:    "local",
		Status:    "running",
		NodeCount: 1,
		Version:   "v1.31.0",
		Config:    map[string]interface{}{},
		Metadata:  map[string]string{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "test-user",
	}
	
	err = stateManager.SaveClusterState(ctx, testCluster)
	if err != nil {
		t.Fatalf("Failed to create test cluster: %v", err)
	}

	testResource := &Resource{
		ID:          clusterName + "-ingress",
		ClusterName: clusterName,
		Type:        "addon",
		Name:        "ingress",
		Namespace:   "kube-system",
		Status:      "enabled",
		Config: map[string]any{
			"controller": "nginx",
			"enabled": true,
		},
		Dependencies: []string{"nginx-controller"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	t.Run("SaveResource", func(t *testing.T) {
		err := stateManager.SaveResource(ctx, testResource)
		if err != nil {
			t.Fatalf("SaveResource() error = %v", err)
		}
	})

	t.Run("GetResource", func(t *testing.T) {
		retrievedResource, err := stateManager.GetResource(ctx, clusterName, testResource.ID)
		if err != nil {
			t.Fatalf("GetResource() error = %v", err)
		}
		
		if retrievedResource == nil {
			t.Fatal("GetResource() returned nil resource")
		}
		
		if retrievedResource.ID != testResource.ID {
			t.Errorf("GetResource() ID = %v, want %v", retrievedResource.ID, testResource.ID)
		}
		
		if retrievedResource.Type != testResource.Type {
			t.Errorf("GetResource() type = %v, want %v", retrievedResource.Type, testResource.Type)
		}
		
		if retrievedResource.Status != testResource.Status {
			t.Errorf("GetResource() status = %v, want %v", retrievedResource.Status, testResource.Status)
		}
	})

	t.Run("ListResources", func(t *testing.T) {
		resources, err := stateManager.ListResources(ctx, clusterName)
		if err != nil {
			t.Fatalf("ListResources() error = %v", err)
		}
		
		if len(resources) == 0 {
			t.Error("ListResources() should return at least one resource")
		}
		
		found := false
		for _, resource := range resources {
			if resource.ID == testResource.ID {
				found = true
				if resource.Status != testResource.Status {
					t.Errorf("ListResources() resource status = %v, want %v", resource.Status, testResource.Status)
				}
				break
			}
		}
		
		if !found {
			t.Error("ListResources() should include test resource")
		}
	})

	t.Run("UpdateResource", func(t *testing.T) {
		testResource.Status = "disabled"
		testResource.UpdatedAt = time.Now()
		
		err := stateManager.SaveResource(ctx, testResource)
		if err != nil {
			t.Fatalf("SaveResource() update error = %v", err)
		}
		
		retrievedResource, err := stateManager.GetResource(ctx, clusterName, testResource.ID)
		if err != nil {
			t.Fatalf("GetResource() after update error = %v", err)
		}
		
		if retrievedResource.Status != "disabled" {
			t.Errorf("GetResource() after update status = %v, want %v", retrievedResource.Status, "disabled")
		}
	})

	t.Run("DeleteResource", func(t *testing.T) {
		err := stateManager.DeleteResource(ctx, clusterName, testResource.ID)
		if err != nil {
			t.Fatalf("DeleteResource() error = %v", err)
		}
		
		retrievedResource, err := stateManager.GetResource(ctx, clusterName, testResource.ID)
		if err != nil {
			t.Fatalf("GetResource() after delete error = %v", err)
		}
		
		if retrievedResource != nil {
			t.Error("GetResource() after delete should return nil")
		}
	})
}

func TestSQLiteStateManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "atlas-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "integration.db")
	stateManager, err := NewSQLiteStateManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateManager.Disconnect(context.Background())

	ctx := context.Background()
	clusterName := "integration-test-cluster"

	t.Run("FullClusterLifecycle", func(t *testing.T) {
		cluster := &ClusterState{
			Name:      clusterName,
			Provider:  "local",
			Region:    "local",
			Status:    "creating",
			NodeCount: 2,
			Version:   "v1.31.0",
			Config: map[string]interface{}{
				"enable_ingress": true,
				"enable_monitoring": false,
			},
			Metadata: map[string]string{
				"created_by_test": "true",
				"integration": "test",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: "integration-test",
		}

		err := stateManager.SaveClusterState(ctx, cluster)
		if err != nil {
			t.Fatalf("SaveClusterState() error = %v", err)
		}

		ingressResource := &Resource{
			ID:          clusterName + "-ingress",
			ClusterName: clusterName,
			Type:        "addon",
			Name:        "ingress",
			Status:      "enabled",
			Config: map[string]any{
				"controller": "nginx",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = stateManager.SaveResource(ctx, ingressResource)
		if err != nil {
			t.Fatalf("SaveResource() error = %v", err)
		}

		cluster.Status = "running"
		cluster.UpdatedAt = time.Now()
		err = stateManager.SaveClusterState(ctx, cluster)
		if err != nil {
			t.Fatalf("SaveClusterState() update error = %v", err)
		}

		retrievedCluster, err := stateManager.GetClusterState(ctx, clusterName)
		if err != nil {
			t.Fatalf("GetClusterState() error = %v", err)
		}

		if retrievedCluster.Status != "running" {
			t.Errorf("Cluster status = %v, want %v", retrievedCluster.Status, "running")
		}

		if len(retrievedCluster.Resources) == 0 {
			t.Error("Cluster should have resources loaded")
		}

		cluster.Status = "stopped"
		cluster.UpdatedAt = time.Now()
		err = stateManager.SaveClusterState(ctx, cluster)
		if err != nil {
			t.Fatalf("SaveClusterState() stop error = %v", err)
		}

		err = stateManager.DeleteResource(ctx, clusterName, ingressResource.ID)
		if err != nil {
			t.Fatalf("DeleteResource() error = %v", err)
		}

		err = stateManager.DeleteClusterState(ctx, clusterName)
		if err != nil {
			t.Fatalf("DeleteClusterState() error = %v", err)
		}

		finalCluster, err := stateManager.GetClusterState(ctx, clusterName)
		if err != nil {
			t.Fatalf("GetClusterState() after cleanup error = %v", err)
		}

		if finalCluster != nil {
			t.Error("Cluster should be deleted")
		}
	})
}

func BenchmarkSQLiteStateManager_SaveClusterState(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "atlas-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "bench.db")
	stateManager, err := NewSQLiteStateManager(dbPath)
	if err != nil {
		b.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateManager.Disconnect(context.Background())

	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cluster := &ClusterState{
			Name:      "bench-cluster",
			Provider:  "local",
			Region:    "local",
			Status:    "running",
			NodeCount: 2,
			Version:   "v1.31.0",
			Config:    map[string]interface{}{},
			Metadata:  map[string]string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: "benchmark",
		}
		
		stateManager.SaveClusterState(ctx, cluster)
		stateManager.DeleteClusterState(ctx, cluster.Name)
	}
}