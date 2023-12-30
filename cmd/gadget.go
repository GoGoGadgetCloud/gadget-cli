package main

import (
	"os"

	"github.com/stefan79/gadget-cli/pkg/commands"
	"github.com/urfave/cli/v2"
)

func main() {

	session := commands.NewSession()
	workActions := commands.NewWorkContext(session)
	bootstrapActions, err := commands.NewBootstrapContext(session)
	if err != nil {
		panic(err)
	}
	deployActions, err := commands.NewDeployContext(session)
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			workActions.CreateCommand(),
			bootstrapActions.CreateCommand(),
			deployActions.CreateCommand(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
