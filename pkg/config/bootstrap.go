package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type (
	DeploymentConfig struct {
		S3BucketName *string
	}

	Bootstrap struct {
		DeploymentConfig
	}
)

func SaveBootstrap(bootstrapPath string, c *Bootstrap) error {

	// Marshal Config struct to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(bootstrapPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadBootstrap(bootstrapPath string) (*Bootstrap, error) {

	// Check if file exists
	if _, err := os.Stat(bootstrapPath); os.IsNotExist(err) {
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(bootstrapPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal YAML to Config struct
	var bootstrap Bootstrap
	err = yaml.Unmarshal(data, &bootstrap)
	if err != nil {
		return nil, err
	}

	return &bootstrap, nil
}
