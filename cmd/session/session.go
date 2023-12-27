package session

import (
	"context"
	"fmt"
	"log"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stefan79/gadget-cli/cmd/config"
	"github.com/stefan79/gadget-cli/cmd/util"
	"github.com/urfave/cli/v2"
)

type DefaultSession struct {
	StagingDirectory string
	Config           *config.Config
}

func NewSession() (*DefaultSession, error) {
	sesh := &DefaultSession{
		StagingDirectory: "./.gadget/staging",
	}
	err := os.MkdirAll(sesh.StagingDirectory, 0777)
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadConfig()
	sesh.Config = cfg
	return sesh, err
}

func (s *DefaultSession) HandleError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

}

func (s *DefaultSession) Deploy(cCtx *cli.Context) error {
	if s.Config == nil {
		return fmt.Errorf("No Config Set, run init first")
	}
	compiledCommand, err := util.CompileLambda(cCtx.Args().First(), s.StagingDirectory)
	if err != nil {
		return err
	}
	xcompiledCommand, err := util.XCompileLambda(cCtx.Args().First(), s.StagingDirectory)
	if err != nil {
		return err
	}
	zippedCommand, err := util.ZipFile(xcompiledCommand, s.StagingDirectory)
	if err != nil {
		return err
	}
	err = util.UploadLambda(zippedCommand, s.StagingDirectory, *s.Config.S3BucketName)
	if err != nil {
		return err
	}
	stackTemplate, err := util.GenerateDeployment(compiledCommand, s.StagingDirectory, *s.Config.S3BucketName)
	if err != nil {
		return err
	}
	client, err := createCloudFormationClient(context.Background())
	if err != nil {
		return err
	}
	return util.DeployStack(client, "gadgetstack", stackTemplate)
}

func (s *DefaultSession) Init(cCtx *cli.Context) error {
	fmt.Println("Will create a new CloudFormation Stack")
	ctx := context.Background()
	client, err := createCloudFormationClient(ctx)
	if err != nil {
		return err
	}
	tmpl, err := util.CreateGadgetTemplate()
	if err != nil {
		return err
	}
	err = util.DeployGatdgetTemplate(ctx, client, "gadget-init", string(tmpl))
	if err != nil {
		return err
	}
	cfg, err := util.GenerateConfig(ctx, client, "gadget-init")
	if err != nil {
		return err
	}
	return config.SaveConfig(&config.Config{
		DeploymentConfig: *cfg,
	})

}

func createCloudFormationClient(ctx context.Context) (*cloudformation.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := cloudformation.NewFromConfig(cfg)
	return client, nil
}
