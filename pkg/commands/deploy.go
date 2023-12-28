package commands

import (
	"context"
	"fmt"

	"github.com/stefan79/gadget-cli/pkg/adapter"
	"github.com/stefan79/gadget-cli/pkg/config"
	"github.com/urfave/cli/v2"
)

type (
	DefaultDeployActions struct {
		Session               *Session
		S3Adapter             adapter.S3Adapter
		CloudFormationAdapter adapter.CloudFormationAdapter
		StagingAdapter        adapter.StagingAdapter
		BootStrap             *config.Bootstrap
	}

	DeployActions interface {
		Deploy(cCtx *cli.Context) error
	}

	DeployContext interface {
		CommandBuilder
		DeployActions
	}
)

func NewDeployContext(session *Session) (DeployContext, error) {
	stagingAdapter, err := adapter.NewStagingAdapter(session.StagingPath)
	if err != nil {
		return nil, err
	}
	s3Adapter, err := adapter.NewS3Adapter()
	if err != nil {
		return nil, err
	}
	bootstrap, err := session.LoadBootstrapConfig()
	fmt.Println("NewDeployContext Bootstrap", bootstrap, "err", err)
	if err != nil {
		return nil, err
	}
	cloudformationAdapter, err := adapter.NewCloudFormationAdapter()
	if err != nil {
		return nil, err
	}
	return &DefaultDeployActions{
		Session:               session,
		StagingAdapter:        stagingAdapter,
		S3Adapter:             s3Adapter,
		BootStrap:             bootstrap,
		CloudFormationAdapter: cloudformationAdapter,
	}, nil
}

func (a *DefaultDeployActions) Deploy(cCtx *cli.Context) error {
	ctx := context.Background()
	inputSource := cCtx.Args().First()
	compiledCommand := "lambda-local"
	err := a.StagingAdapter.Compile(&inputSource, &compiledCommand)
	if err != nil {
		return err
	}
	xcompiledCommand := "lambda-remote"
	options := make(map[string]string)
	options["GOOS"] = "linux"
	options["GOARCH"] = "amd64"
	err = a.StagingAdapter.CompileWithOptions(&inputSource, &xcompiledCommand, options)
	if err != nil {
		return err
	}
	fullxcompiledCommand, err := a.StagingAdapter.GetFileFromStaging(&xcompiledCommand)
	if err != nil {
		return err
	}
	zipFileName := "lambda-remote.zip"
	err = a.StagingAdapter.Zip(fullxcompiledCommand, &zipFileName)
	if err != nil {
		return err
	}
	fullZipFileName, err := a.StagingAdapter.GetFileFromStaging(&zipFileName)
	if err != nil {
		return err
	}
	bucketKey := "bootstrap.zip"
	fmt.Println("BootStrap", *a.BootStrap)
	err = a.S3Adapter.UploadFile(ctx, *fullZipFileName, *a.BootStrap.S3BucketName, bucketKey)
	if err != nil {
		return err
	}
	cloudformationName := "cloudformation.yaml"
	fullCompiledCommand, err := a.StagingAdapter.GetFileFromStaging(&compiledCommand)
	if err != nil {
		return err
	}
	err = a.StagingAdapter.GenerateTemplate(fullCompiledCommand, &cloudformationName, &xcompiledCommand, a.BootStrap.S3BucketName, &bucketKey)
	if err != nil {
		return err
	}
	fullCloudformationName, err := a.StagingAdapter.GetFileFromStaging(&cloudformationName)
	if err != nil {
		return err
	}
	return a.CloudFormationAdapter.DeployTemplateAsFile(ctx, "gadgeto", *fullCloudformationName)
}

func (a *DefaultDeployActions) CreateCommand() *cli.Command {
	return &cli.Command{
		Name:   "deploy",
		Usage:  "Deploy the lambda function",
		Action: a.Deploy,
	}
}
