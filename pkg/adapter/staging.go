package adapter

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type (
	DefaultStagingAdapter struct {
		StagingArea *string
	}
	StagingAdapter interface {
		GetFileFromStaging(base *string) (*string, error)
		Compile(inputSource *string, outputTarget *string) error
		GenerateTemplate(inputSource *string, outputTarget *string, handlerName *string, s3Bucket *string, s3Key *string) error
		CompileWithOptions(inputSource *string, outputTarget *string, options map[string]string) error
		Zip(inputSource *string, outputTarget *string) error
	}
)

func NewStagingAdapter(stagingArea *string) (StagingAdapter, error) {
	return &DefaultStagingAdapter{
		StagingArea: stagingArea,
	}, nil
}

func (a *DefaultStagingAdapter) GetFileFromStaging(fileName *string) (*string, error) {
	fullPath := filepath.Join(*a.StagingArea, *fileName)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist in staging area", *fileName)
	}

	return &fullPath, nil
}

func (a *DefaultStagingAdapter) Compile(inputSource *string, outputTarget *string) error {
	targetFile, err := createFullPathReference(*outputTarget, *a.StagingArea)
	if err != nil {
		return err
	}
	options := make(map[string]string)
	err = runCommand("go", options, "build", "-o", targetFile, *inputSource)
	return err
}

func (a *DefaultStagingAdapter) CompileWithOptions(inputSource *string, outputTarget *string, options map[string]string) error {
	targetFile, err := createFullPathReference(*outputTarget, *a.StagingArea)
	if err != nil {
		return err
	}
	err = runCommand("go", options, "build", "-o", targetFile, *inputSource)
	return err
}

func (a *DefaultStagingAdapter) Zip(inputSource *string, outputTarget *string) error {
	targetFile, err := createFullPathReference(*outputTarget, *a.StagingArea)
	if err != nil {
		return err
	}
	zipFile, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Open source file
	fileToZip, err := os.Open(*inputSource)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Create a writer for the file
	writer, err := zipWriter.Create(filepath.Base(*inputSource))
	if err != nil {
		return err
	}

	// Copy the file content to the zip archive
	_, err = io.Copy(writer, fileToZip)
	return err
}

func (a *DefaultStagingAdapter) GenerateTemplate(inputSource *string, outputTarget *string, handlerName *string, s3bucket *string, s3key *string) error {
	targetFile, err := createFullPathReference(*outputTarget, *a.StagingArea)
	if err != nil {
		return err
	}
	err = runCommand(*inputSource, nil, "deployment", "generate", "--template", targetFile, "--s3bucket", *s3bucket, "--s3key", *s3key, "--handler", *handlerName)
	return err
}

func createFullPathReference(basefile string, path string) (string, error) {
	if basefile != filepath.Base(basefile) {
		return "", fmt.Errorf("basefile must be a relative file ")
	}
	return filepath.Join(path, basefile), nil

}

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
