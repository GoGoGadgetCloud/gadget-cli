package commands

import (
	"path/filepath"

	"github.com/stefan79/gadget-cli/pkg/config"
	"github.com/urfave/cli/v2"
)

type (
	Session struct {
		ApplicationConfigPath *string
		WorkPath              *string
		StagingPath           *string
	}

	CommandBuilder interface {
		CreateCommand() *cli.Command
	}
)

func NewSession() *Session {
	defaultPath := "./gadget.yaml"
	defaultWorkPath := "./.gadget"
	defaultStagingPath := "./.gadget/staging"
	return &Session{
		ApplicationConfigPath: &defaultPath,
		WorkPath:              &defaultWorkPath,
		StagingPath:           &defaultStagingPath,
	}
}

func (s *Session) LoadApplicationConfig() (*config.ApplicationConfig, error) {
	conf, err := config.LoadConfig(*s.ApplicationConfigPath)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (s *Session) SaveApplicationConfig(conf *config.ApplicationConfig) error {
	return config.SaveConfig(conf, *s.ApplicationConfigPath)
}

func (s *Session) LoadBootstrapConfig() (*config.Bootstrap, error) {
	bootstrap, err := config.LoadBootstrap(filepath.Join(*s.WorkPath, "bootstrap.yaml"))
	if err != nil {
		return nil, err
	}
	return bootstrap, nil
}

func (s *Session) SaveBootstrapConfig(bootstrap *config.Bootstrap) error {
	return config.SaveBootstrap(filepath.Join(*s.WorkPath, "bootstrap.yaml"), bootstrap)
}
