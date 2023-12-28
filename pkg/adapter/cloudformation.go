package adapter

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type (
	CloudFormationSDK struct {
		Client *cloudformation.Client
	}
	CloudFormationAdapter interface {
		DeployTemplateAsBytes(ctx context.Context, name string, data []byte) error
		DeployTemplateAsFile(ctx context.Context, name string, file string) error
		LoadTemplate(ctx context.Context, name string) ([]byte, error)
	}
)

func NewCloudFormationAdapter() (CloudFormationAdapter, error) {
	client, err := createCloudFormationClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &CloudFormationSDK{
		Client: client,
	}, nil
}

func (c *CloudFormationSDK) LoadTemplate(ctx context.Context, name string) ([]byte, error) {
	resp, err := c.Client.GetTemplate(ctx, &cloudformation.GetTemplateInput{
		StackName: &name,
	})
	if err != nil {
		return nil, err
	}
	return []byte(*resp.TemplateBody), nil
}

func (c *CloudFormationSDK) DeployTemplateAsFile(ctx context.Context, name string, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return c.DeployTemplateAsBytes(ctx, name, data)
}

func (c *CloudFormationSDK) DeployTemplateAsBytes(ctx context.Context, name string, data []byte) error {
	bodyAsString := string(data)
	_, err := c.Client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    &name,
		TemplateBody: &bodyAsString,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
		},
	})
	if err != nil {
		return err
	}

	for {
		resp, err := c.Client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: &name,
		})
		if err != nil {
			return err
		}

		if len(resp.Stacks) == 0 {
			return fmt.Errorf("stack %s not found", name)
		}

		switch resp.Stacks[0].StackStatus {
		case types.StackStatusCreateComplete:
			return nil
		case types.StackStatusCreateInProgress:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("creation of stack %s failed with status %s", name, resp.Stacks[0].StackStatus)
		}
	}
}

func createCloudFormationClient(ctx context.Context) (*cloudformation.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := cloudformation.NewFromConfig(cfg)
	return client, nil
}
