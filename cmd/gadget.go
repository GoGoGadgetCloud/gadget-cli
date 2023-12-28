package main

import (
	"fmt"
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
	for _, cmd := range app.Commands {
		fmt.Println("Command", cmd.Name)
	}
	fmt.Println("App Commands", app.Commands)
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
