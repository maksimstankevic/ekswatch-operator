package controller

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type TestCluster struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}

type TestClustersFile struct {
	TestClusters []TestCluster `yaml:"clusters"`
}

func TestAppendSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		secrets  []string
		expected []string
	}{
		{
			name:     "Replace NoSecretsFound",
			secret:   "mysecret",
			secrets:  []string{"NoSecretsFound"},
			expected: []string{"mysecret"},
		},
		{
			name:     "Append to existing secrets",
			secret:   "anothersecret",
			secrets:  []string{"secret1"},
			expected: []string{"secret1", "anothersecret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendSecret(tt.secret, tt.secrets)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddClusterIfNotExists(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "clusters-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	clusters := TestClustersFile{
		TestClusters: []TestCluster{
			{Name: "existing-cluster", Labels: map[string]string{"env": "prod"}},
		},
	}
	data, _ := yaml.Marshal(&clusters)
	os.WriteFile(tmpFile.Name(), data, 0644)
	assert.FileExists(t, tmpFile.Name())
	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(content), "existing-cluster")

	ctx := context.TODO()

	// Test adding a new cluster
	err = AddClusterIfNotExists(tmpFile.Name(), "new-cluster", ctx)
	assert.NoError(t, err)

	// Check file contents
	updated, _ := os.ReadFile(tmpFile.Name())
	var updatedClusters TestClustersFile
	yaml.Unmarshal(updated, &updatedClusters)
	assert.Len(t, updatedClusters.TestClusters, 2)
	assert.Equal(t, "new-cluster", updatedClusters.TestClusters[1].Name)

	// Test adding an existing cluster (should not duplicate)
	err = AddClusterIfNotExists(tmpFile.Name(), "existing-cluster", ctx)
	assert.NoError(t, err)
	updated, _ = os.ReadFile(tmpFile.Name())
	yaml.Unmarshal(updated, &updatedClusters)
	assert.Len(t, updatedClusters.TestClusters, 2)
}

func TestAddClusterIfNotExists_FileNotFound(t *testing.T) {
	ctx := context.TODO()
	err := AddClusterIfNotExists("nonexistent.yaml", "test-cluster", ctx)
	assert.Error(t, err)
}

func TestAddClusterIfNotExists_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	os.WriteFile(tmpFile.Name(), []byte("not: [valid"), 0644)
	ctx := context.TODO()
	err = AddClusterIfNotExists(tmpFile.Name(), "test-cluster", ctx)
	assert.Error(t, err)
}

func TestMatchesAnyRegex(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name      string
		regexList []string
		cluster   string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "Match found",
			regexList: []string{"^prod-.*", "^dev-.*"},
			cluster:   "prod-cluster",
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "No match",
			regexList: []string{"^dev-.*"},
			cluster:   "prod-cluster",
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "Invalid regex",
			regexList: []string{"[invalid"},
			cluster:   "prod-cluster",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "Empty regex list",
			regexList: []string{},
			cluster:   "prod-cluster",
			wantMatch: false,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchesAnyRegex(tt.regexList, tt.cluster, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMatch, got)
			}
		})
	}
}
