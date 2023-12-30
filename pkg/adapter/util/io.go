package util

import (
	"os"

	"gopkg.in/yaml.v2"
)

func ReadYAMLFile(filePath string) (interface{}, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal YAML to interface{}
	var result interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func SaveYAMLFile(filePath string, data interface{}) error {
	// Marshal interface{} to YAML
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		return err
	}

	return nil
}
