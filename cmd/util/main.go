package util

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func runCommand(name string, env map[string]string, args ...string) error {

	command := exec.Command(name, args...)
	fmt.Println("Running command", command.String())
	if errors.Is(command.Err, exec.ErrDot) {
		command.Err = nil
	}
	var sdterr bytes.Buffer
	command.Stderr = &sdterr
	var sdtout bytes.Buffer
	command.Stdout = &sdtout

	command.Env = os.Environ()
	if env != nil {
		for k, v := range env {
			command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if err := command.Run(); err != nil {
		fmt.Println(sdtout.String())
		return fmt.Errorf("failed command: %s, %s", command.String(), sdterr.String())
	}
	return nil

}

func CompileLambda(sourceFile string, stagingDirectory string) (string, error) {
	targetFile := createFileInStagingArea("lambda", stagingDirectory)
	err := runCommand("go", nil, "build", "-o", targetFile, sourceFile)
	return targetFile, err
}

func XCompileLambda(sourceFile string, stagingDirectory string) (string, error) {
	targetFile := createFileInStagingArea("remote", stagingDirectory)
	env := make(map[string]string)
	env["GOOS"] = "linux"
	env["GOARCH"] = "amd64"
	err := runCommand("go", env, "build", "-o", targetFile, sourceFile)
	return targetFile, err
}

func ZipFile(sourceFile string, stagingDirectory string) (string, error) {
	// Create target zip file
	targetFile := filepath.Join(stagingDirectory, filepath.Base(sourceFile)+".zip")
	zipFile, err := os.Create(targetFile)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Open source file
	fileToZip, err := os.Open(sourceFile)
	if err != nil {
		return "", err
	}
	defer fileToZip.Close()

	// Create a writer for the file
	writer, err := zipWriter.Create(filepath.Base(sourceFile))
	if err != nil {
		return "", err
	}

	// Copy the file content to the zip archive
	_, err = io.Copy(writer, fileToZip)
	if err != nil {
		return "", err
	}

	return targetFile, nil
}

func UploadLambda(commandFile string, stagingDirectory string, bucketName string) error {
	client, err := createS3Client()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(commandFile)
	if err != nil {
		return err
	}
	key := "lambda.zip"
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return err
	}
	return nil
}

func DeployStack(client *cloudformation.Client, stackName string, templateFile string) error {
	// Load template from file
	template, err := os.ReadFile(templateFile)
	if err != nil {
		return err
	}

	// Check if stack exists
	describeInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	_, err = client.DescribeStacks(context.TODO(), describeInput)
	stackExists := err == nil

	// Create or update stack
	if stackExists {
		// Update stack
		updateInput := &cloudformation.UpdateStackInput{
			StackName:    aws.String(stackName),
			TemplateBody: aws.String(string(template)),
			Capabilities: []types.Capability{
				types.CapabilityCapabilityIam,
			},
		}
		_, err = client.UpdateStack(context.TODO(), updateInput)
	} else {
		// Create stack
		createInput := &cloudformation.CreateStackInput{
			StackName:    aws.String(stackName),
			TemplateBody: aws.String(string(template)),
			Capabilities: []types.Capability{
				types.CapabilityCapabilityIam,
			},
		}
		_, err = client.CreateStack(context.TODO(), createInput)
	}
	if err != nil {
		return err
	}

	// Wait for stack to be created or updated
	waitInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	err = waitUntilStackCreateComplete(client, waitInput)
	return err
}

func waitUntilStackCreateComplete(client *cloudformation.Client, input *cloudformation.DescribeStacksInput) error {
	for {
		resp, err := client.DescribeStacks(context.TODO(), input)
		if err != nil {
			return err
		}

		if len(resp.Stacks) == 0 {
			return fmt.Errorf("stack not found")
		}

		switch resp.Stacks[0].StackStatus {
		case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
			return nil
		case types.StackStatusCreateFailed, types.StackStatusRollbackFailed, types.StackStatusDeleteFailed, types.StackStatusUpdateRollbackFailed:
			return fmt.Errorf("stack creation failed")
		default:
			time.Sleep(time.Second * 5)
		}
	}
}

func createS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return client, nil
}

func GenerateDeployment(commandFile string, stagingDirectory string, bucketName string) (string, error) {
	targetFile := createFileInStagingArea("cloudformation.yaml", stagingDirectory)
	err := runCommand(commandFile, nil, "deploy", "-o", targetFile, "-b", bucketName)
	return targetFile, err
}

func createFileInStagingArea(path string, stagingDirectory string) string {
	return filepath.Join(stagingDirectory, path)
}
