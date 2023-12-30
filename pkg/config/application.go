package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type (
	ApplicationConfig struct {
		Name     *string
		Commands []*Command
		Tags     map[string]string
	}

	Command struct {
		Name *string
		Path *string
	}
)

func SaveConfig(ac *ApplicationConfig, filePath string) error {
	// Marshal ApplicationConfig struct to YAML
	data, err := yaml.Marshal(ac)
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0777)
	if err != nil {
		return err
	}

	return nil
}

func (ac *ApplicationConfig) AddCommand(cmd *Command) error {
	for _, existingCmd := range ac.Commands {
		if existingCmd.Name == cmd.Name {
			return fmt.Errorf("command %s already exists", *cmd.Name)
		}
	}

	ac.Commands = append(ac.Commands, cmd)
	return nil
}

func (ac *ApplicationConfig) SetTag(key string, value string) error {
	if ac.Tags == nil {
		ac.Tags = make(map[string]string)
	}

	ac.Tags[key] = value
	return nil
}

func LoadConfig(filePath string) (*ApplicationConfig, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal YAML to ApplicationConfig struct
	var ac ApplicationConfig
	err = yaml.Unmarshal(data, &ac)
	if err != nil {
		return nil, err
	}

	return &ac, nil
}
