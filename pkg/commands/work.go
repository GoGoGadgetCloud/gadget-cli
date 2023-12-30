package commands

import (
	"path/filepath"
	"strings"

	"github.com/stefan79/gadget-cli/pkg/config"
	"github.com/urfave/cli/v2"
)

type (
	DefaultWorkActions struct {
		Session *Session
	}

	WorkActions interface {
		Init(cCtx *cli.Context) error
		Use(cCtx *cli.Context) error
		SetTag(cCtx *cli.Context) error
	}

	WorkContext interface {
		CommandBuilder
		WorkActions
	}
)

func NewWorkContext(session *Session) WorkContext {
	return &DefaultWorkActions{
		Session: session,
	}
}

func (a *DefaultWorkActions) SetTag(cCtx *cli.Context) error {
	key := cCtx.String("key")
	value := cCtx.String("value")
	conf, err := a.Session.LoadApplicationConfig()
	if err != nil {
		return err
	}
	err = conf.SetTag(key, value)
	if err != nil {
		return err
	}
	return a.Session.SaveApplicationConfig(conf)
}

func (a *DefaultWorkActions) Init(cCtx *cli.Context) error {
	applicationName := cCtx.Args().First()
	conf := &config.ApplicationConfig{
		Name:     &applicationName,
		Commands: make([]*config.Command, 0),
	}
	return a.Session.SaveApplicationConfig(conf)
}

func (a *DefaultWorkActions) Use(cCtx *cli.Context) error {
	conf, err := a.Session.LoadApplicationConfig()
	if err != nil {
		return err
	}
	path := cCtx.Args().First()
	baseName := filepath.Base(path)
	name := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	cmd := &config.Command{
		Name: &name,
		Path: &path,
	}

	err = conf.AddCommand(cmd)
	if err != nil {
		return err
	}
	return a.Session.SaveApplicationConfig(conf)
}

func (a *DefaultWorkActions) CreateCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "work",
		Usage: "Manages your gadget workspace",
		Subcommands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "initialize your app",
				Args:   true,
				Action: a.Init,
			},
			{
				Name:   "use",
				Usage:  "use a command in your app",
				Args:   true,
				Action: a.Use,
			},
			{
				Name:   "setTag",
				Usage:  "add a tag to your app",
				Args:   true,
				Action: a.SetTag,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "key",
						Usage:    "key of the tag",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "value",
						Usage:    "value of the tag",
						Required: true,
					},
				},
			},
		},
	}
	return cmd
}
