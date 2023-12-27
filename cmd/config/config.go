package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type (
	DeploymentConfig struct {
		S3BucketName *string
	}

	Config struct {
		DeploymentConfig
	}
)

func SaveConfig(c *Config) error {
	configPath := "./.gadget/config.yaml"

	// Marshal Config struct to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfig() (*Config, error) {
	configPath := "./.gadget/config.yaml"

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal YAML to Config struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
