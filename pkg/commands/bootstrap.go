package commands

import (
	"context"
	"fmt"

	//"github.com/aws/aws-sdk-go/service/s3"
	"github.com/awslabs/goformation/v7"
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/stefan79/gadget-cli/pkg/adapter"
	"github.com/stefan79/gadget-cli/pkg/config"
	"github.com/urfave/cli/v2"
)

type (
	DefaultBootstrapActions struct {
		Session               *Session
		CloudformationAdapter adapter.CloudFormationAdapter
	}

	BootstrapActions interface {
		Init(cCtx *cli.Context) error
	}

	BootstrapContext interface {
		CommandBuilder
		BootstrapActions
	}
)

// CreateCommand implements BootstrapContext.
func (a *DefaultBootstrapActions) CreateCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "bootstrap",
		Usage:  "initialize a configuration ",
		Action: a.Init,
	}
	return cmd

}

func NewBootstrapContext(session *Session) (BootstrapContext, error) {
	cloudformationAdapter, err := adapter.NewCloudFormationAdapter()
	if err != nil {
		return nil, err
	}
	return &DefaultBootstrapActions{
		Session:               session,
		CloudformationAdapter: cloudformationAdapter,
	}, nil

}

func (a *DefaultBootstrapActions) Init(cCtx *cli.Context) error {
	fmt.Println("Will create a new CloudFormation Stack")
	ctx := context.Background()
	tmpl, err := createGadgetTemplate()
	if err != nil {
		return err
	}
	err = a.CloudformationAdapter.DeployTemplateAsBytes(ctx, "gadget-init", tmpl)
	if err != nil {
		return err
	}
	templateAsBytes, err := a.CloudformationAdapter.LoadTemplate(ctx, "gadget-init")
	if err != nil {
		return err
	}
	cfg, err := generateBootstrap(ctx, templateAsBytes)
	if err != nil {
		return err
	}
	return a.Session.SaveBootstrapConfig(&config.Bootstrap{
		DeploymentConfig: *cfg,
	})
}

func generateBootstrap(ctx context.Context, templateAsBytes []byte) (*config.DeploymentConfig, error) {
	template, err := goformation.ParseYAML(templateAsBytes)
	if err != nil {
		return nil, err
	}

	cfg := &config.Bootstrap{}

	for _, output := range template.Outputs {
		switch output.Export.Name {
		case "DeployBucket":
			if bucketReference, ok := output.Value.(string); ok {
				cfg.S3BucketName = &bucketReference
			} else {
				return nil, fmt.Errorf("stack does not have DeployBucket output which is not a string")
			}
		}

	}
	if cfg.S3BucketName == nil {
		return nil, fmt.Errorf("stack does not have a DeployBucket output")
	}

	return &cfg.DeploymentConfig, nil
}

func createGadgetTemplate() ([]byte, error) {
	template := cloudformation.NewTemplate()
	template.Resources["DeployBucket"] = &s3.Bucket{}

	template.Outputs["DeployBucket"] = cloudformation.Output{
		Value: cloudformation.Ref("DeployBucket"),
	}

	return template.YAML()
}
