package controller

import (
	"context"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func appendSecret(secret string, secrets []string) []string {
	if secrets[0] == "NoSecretsFound" {
		secrets[0] = secret
	} else {
		secrets = append(secrets, secret)
	}
	return secrets
}

// AddClusterIfNotExists checks if a cluster exists in the YAML file and adds it if not.
func AddClusterIfNotExists(filePath string, clusterName string, ctx context.Context) error {

	logging := log.FromContext(ctx)

	// Open the YAML file
	file, err := os.ReadFile(filePath)
	if err != nil {
		logging.Error(err, "failed to read file")
		return err
	}

	// Parse the YAML content
	var clustersFile ClustersFile
	err = yaml.Unmarshal(file, &clustersFile)
	if err != nil {
		logging.Error(err, "failed to parse YAML")
		return err
	}

	// Check if the cluster already exists
	for _, cluster := range clustersFile.Clusters {
		if cluster.Name == clusterName {
			logging.Info("Cluster already exists in the YAML file.")
			return nil
		}
	}

	// Add the new cluster
	newCluster := Cluster{
		Name: clusterName,
		Labels: map[string]string{
			"env": "FromController", // Example label, modify as needed
		},
	}
	clustersFile.Clusters = append(clustersFile.Clusters, newCluster)

	// Marshal the updated content back to YAML
	updatedYAML, err := yaml.Marshal(&clustersFile)
	if err != nil {
		logging.Error(err, "failed to marshal updated YAML")
		return err
	}

	// Write the updated YAML back to the file
	err = os.WriteFile(filePath, updatedYAML, 0644)
	if err != nil {
		logging.Error(err, "failed to write updated YAML to file")
		return err
	}

	logging.Info("Cluster added successfully.")
	return nil
}

// MatchesAnyRegex checks if the clusterName matches any of the provided regex patterns.
func MatchesAnyRegex(regexList []string, clusterName string, ctx context.Context) (bool, error) {

	logging := log.FromContext(ctx)

	for _, pattern := range regexList {
		// Compile the regex pattern
		re, err := regexp.Compile(pattern)
		if err != nil {
			logging.Error(err, "invalid regex pattern: %s", pattern)
			return false, err
		}

		// Check if the clusterName matches the regex
		if re.MatchString(clusterName) {
			return true, nil
		}
	}
	return false, nil
}
