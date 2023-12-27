package main

import (
	"os"

	"github.com/stefan79/gadget-cli/cmd/session"
	"github.com/urfave/cli/v2"
)

func main() {
	sesh, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "deploy",
				Usage:  "deploy a lambda",
				Action: sesh.Deploy,
			},
			{
				Name:   "init",
				Usage:  "initialize a configuration ",
				Action: sesh.Init,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		sesh.HandleError(err)
	}
}
