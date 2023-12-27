package util

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/stefan79/gadget-cli/cmd/config"
)

func DeployGatdgetTemplate(ctx context.Context, c *cloudformation.Client, stackName string, body string) error {
	_, err := c.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    &stackName,
		TemplateBody: &body,
	})
	if err != nil {
		return err
	}

	for {
		resp, err := c.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: &stackName,
		})
		if err != nil {
			return err
		}

		if len(resp.Stacks) == 0 {
			return fmt.Errorf("stack %s not found", stackName)
		}

		switch resp.Stacks[0].StackStatus {
		case types.StackStatusCreateComplete:
			return nil
		case types.StackStatusCreateInProgress:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("creation of stack %s failed with status %s", stackName, resp.Stacks[0].StackStatus)
		}
	}
}

func CreateGadgetTemplate() ([]byte, error) {
	template := goformation.NewTemplate()
	template.Resources["DeployBucket"] = &s3.Bucket{}

	template.Outputs["DeployBucket"] = goformation.Output{
		Value: goformation.Ref("DeployBucket"),
	}

	return template.YAML()
}

func GenerateConfig(ctx context.Context, c *cloudformation.Client, stackName string) (*config.DeploymentConfig, error) {
	resp, err := c.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	if err != nil {
		return nil, err
	}

	if len(resp.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stackName)
	}

	cfg := &config.Config{}
	for _, output := range resp.Stacks[0].Outputs {
		switch *output.OutputKey {
		case "DeployBucket":
			cfg.S3BucketName = output.OutputValue
		}

	}
	if cfg.S3BucketName == nil {
		return nil, fmt.Errorf("stack %s does not have a DeployBucket output", stackName)
	}

	return &cfg.DeploymentConfig, nil
}
