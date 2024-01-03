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
	DeploymentStatus struct {
		Found      bool
		Successful bool
		Status     string
	}

	CloudFormationSDK struct {
		Client *cloudformation.Client
	}
	CloudFormationAdapter interface {
		DeployTemplateAsBytes(ctx context.Context, name string, data []byte) error
		DeployTemplateAsFile(ctx context.Context, name string, file string) error
		UpdateTemplateAsFile(ctx context.Context, name string, file string) error
		GetDeploymentStatus(ctx context.Context, name string) (*DeploymentStatus, error)
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

func (c *CloudFormationSDK) GetDeploymentStatus(ctx context.Context, name string) (*DeploymentStatus, error) {
	resp, err := c.Client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: &name,
	})
	if err != nil {
		if _, ok := err.(*types.StackInstanceNotFoundException); ok {
			return &DeploymentStatus{
				Found:      false,
				Successful: false,
				Status:     "NotFound",
			}, nil
		} else {
			return nil, err
		}
	}
	successful := false
	switch resp.Stacks[0].StackStatus {
	case types.StackStatusCreateComplete:
		successful = true
	}
	return &DeploymentStatus{
		Found:      true,
		Successful: successful,
		Status:     string(resp.Stacks[0].StackStatus),
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

func (c *CloudFormationSDK) UpdateTemplateAsFile(ctx context.Context, name string, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return c.UpdateTemplateAsBytes(ctx, name, data)

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
		case types.StackStatusUpdateCompleteCleanupInProgress:
			time.Sleep(10 * time.Second)
		case types.StackStatusCreateInProgress:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("creation of stack %s failed with status %s", name, resp.Stacks[0].StackStatus)
		}
	}
}

func (c *CloudFormationSDK) UpdateTemplateAsBytes(ctx context.Context, name string, data []byte) error {
	bodyAsString := string(data)
	_, err := c.Client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
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
		case types.StackStatusUpdateComplete:
			return nil
		case types.StackStatusUpdateInProgress:
			time.Sleep(10 * time.Second)
		case types.StackStatusUpdateCompleteCleanupInProgress:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("update of stack %s failed with status %s", name, resp.Stacks[0].StackStatus)
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
