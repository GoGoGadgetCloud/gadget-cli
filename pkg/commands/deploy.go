package commands

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/stefan79/gadget-cli/pkg/adapter"
	"github.com/stefan79/gadget-cli/pkg/config"
	"github.com/urfave/cli/v2"
)

type (
	DefaultDeployActions struct {
		Session                 *Session
		S3Adapter               adapter.S3Adapter
		CloudFormationAdapter   adapter.CloudFormationAdapter
		GadgetoFormationAdapter adapter.GadgetoFormationAdapter
		StagingAdapter          adapter.StagingAdapter
		BootStrap               *config.Bootstrap
		ApplicationConfig       *config.ApplicationConfig
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
	applicationConfig, err := session.LoadApplicationConfig()
	if err != nil {
		return nil, err
	}
	stagingAdapter, err := adapter.NewStagingAdapter(applicationConfig.Name, session.StagingPath)
	if err != nil {
		return nil, err
	}
	s3Adapter, err := adapter.NewS3Adapter()
	if err != nil {
		return nil, err
	}
	bootstrap, err := session.LoadBootstrapConfig()
	if err != nil {
		return nil, err
	}
	cloudformationAdapter, err := adapter.NewCloudFormationAdapter()
	if err != nil {
		return nil, err
	}
	gadgetoFormationAdapter, err := adapter.NewGadgetoFormationAdapter(applicationConfig.Name, applicationConfig.Tags, session.StagingPath)
	if err != nil {
		return nil, err
	}
	return &DefaultDeployActions{
		Session:                 session,
		StagingAdapter:          stagingAdapter,
		S3Adapter:               s3Adapter,
		BootStrap:               bootstrap,
		CloudFormationAdapter:   cloudformationAdapter,
		GadgetoFormationAdapter: gadgetoFormationAdapter,
		ApplicationConfig:       applicationConfig,
	}, nil
}

func (a *DefaultDeployActions) Deploy(cCtx *cli.Context) error {
	ctx := context.Background()
	a.Session.StdOut.Info("Deploying application", "appication", *a.ApplicationConfig.Name)
	for _, command := range a.ApplicationConfig.Commands {
		param := prepareCmdDeploymentParam{
			cmdName:        *command.Name,
			srcFile:        *command.Path,
			bucketName:     *a.BootStrap.S3BucketName,
			stagingAdapter: a.StagingAdapter,
			s3Adapter:      a.S3Adapter,
		}
		a.Session.StdOut.Info("Deploying command", "command", *command.Name)
		cloudformationTemplate, err := prepareCmdDeployment(ctx, param, *a.Session.StdOut)
		if err != nil {
			return fmt.Errorf("error preparing command deployment: %w", err)
		}
		a.Session.StdOut.Debug("Merging command template", "command", *command.Name)
		err = a.GadgetoFormationAdapter.MergeCommandTemplate(command.Name, command.Path, cloudformationTemplate)
		if err != nil {
			return fmt.Errorf("error merging command template: %w", err)
		}
	}

	templateName := "cloudformation.yaml"
	a.Session.StdOut.Debug("Saving application template", "templateName", templateName)
	err := a.GadgetoFormationAdapter.SaveApplicationTemplate(&templateName)
	if err != nil {
		return fmt.Errorf("error saving application template: %w", err)
	}
	fullTemplateName, err := a.StagingAdapter.GetFileFromStaging(&templateName)
	if err != nil {
		return err
	}
	a.Session.StdOut.Debug("Checking Deployment Status", "stackName", *a.ApplicationConfig.Name)
	status, err := a.CloudFormationAdapter.GetDeploymentStatus(ctx, *a.ApplicationConfig.Name)
	if err != nil {
		return fmt.Errorf("error getting deployment status: %w", err)
	}
	a.Session.StdOut.Debug("Deployment Status", "status", status.Status, "found", status.Found, "successful", status.Found)
	if status.Found {
		a.Session.StdOut.Debug("Updating application template", "templateName", *fullTemplateName)
		return a.CloudFormationAdapter.UpdateTemplateAsFile(ctx, *a.ApplicationConfig.Name, *fullTemplateName)
	} else {
		a.Session.StdOut.Debug("Deploying application template", "templateName", *fullTemplateName)
		return a.CloudFormationAdapter.DeployTemplateAsFile(ctx, *a.ApplicationConfig.Name, *fullTemplateName)
	}
}

type prepareCmdDeploymentParam struct {
	cmdName        string
	srcFile        string
	bucketName     string
	stagingAdapter adapter.StagingAdapter
	s3Adapter      adapter.S3Adapter
}

func prepareCmdDeployment(ctx context.Context, param prepareCmdDeploymentParam, logger log.Logger) (*string, error) {
	inputSource := param.srcFile
	compiledCommand := param.cmdName + "_local"
	logger.Debug("Compiling command", "command", param.cmdName)
	err := param.stagingAdapter.Compile(&inputSource, &compiledCommand)
	if err != nil {
		return nil, err
	}
	xcompiledCommand := param.cmdName
	options := make(map[string]string)
	options["GOOS"] = "linux"
	options["GOARCH"] = "amd64"
	logger.Debug("Cross compiling command", "command", xcompiledCommand)
	err = param.stagingAdapter.CompileWithOptions(&inputSource, &xcompiledCommand, options)
	if err != nil {
		return nil, err
	}
	fullxcompiledCommand, err := param.stagingAdapter.GetFileFromStaging(&xcompiledCommand)
	if err != nil {
		return nil, err
	}
	zipFileName := param.cmdName + ".zip"
	logger.Debug("Zipping command", "zipfile", zipFileName)
	err = param.stagingAdapter.Zip(fullxcompiledCommand, &zipFileName)
	if err != nil {
		return nil, err
	}
	fullZipFileName, err := param.stagingAdapter.GetFileFromStaging(&zipFileName)
	//_, err = param.stagingAdapter.GetFileFromStaging(&zipFileName)
	if err != nil {
		return nil, err
	}

	checksum, err := param.stagingAdapter.CalculateCheckSum(&xcompiledCommand)
	if err != nil {
		return nil, fmt.Errorf("error calculating checksum: %w", err)
	}
	logger.Debug("Generate ZIP checksum", "checksum", checksum)
	bucketKey := param.cmdName + "/bootstrap.zip"
	logger.Debug("Uploading Checksum")
	err = param.s3Adapter.CreateFile(ctx, checksum, param.bucketName, bucketKey+".sha256")
	if err != nil {
		return nil, fmt.Errorf("error uploading checksum file %s to bucket %s: %w", checksum, param.bucketName, err)
	}
	logger.Debug("Uploading command", "bucket", param.bucketName, "key", bucketKey)
	err = param.s3Adapter.UploadFile(ctx, *fullZipFileName, param.bucketName, bucketKey)
	if err != nil {
		return nil, fmt.Errorf("error uploading file %s to bucket %s: %w", *fullZipFileName, param.bucketName, err)
	}
	//TODO: Replace region!!!
	s3Url := fmt.Sprintf("https://%s.s3.eu-central-1.amazonaws.com/%s", param.bucketName, bucketKey)
	logger.Debug("Uploaded to S3", "url", s3Url)
	cloudformationName := param.cmdName + "_cf.yaml"
	fullCompiledCommand, err := param.stagingAdapter.GetFileFromStaging(&compiledCommand)
	if err != nil {
		return nil, err
	}
	logger.Debug("Generating command template", "template", cloudformationName)
	err = param.stagingAdapter.GenerateTemplate(fullCompiledCommand, &cloudformationName, &xcompiledCommand, &param.bucketName, &bucketKey)
	if err != nil {
		return nil, err
	}
	return param.stagingAdapter.GetFileFromStaging(&cloudformationName)

}

func (a *DefaultDeployActions) CreateCommand() *cli.Command {
	return &cli.Command{
		Name:   "deploy",
		Usage:  "Deploys the workspace / individual commands to AWS",
		Action: a.Deploy,
	}
}
