package adapter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stefan79/gadget-cli/pkg/adapter/util"
	"golang.org/x/mod/modfile"
)

type (
	GadgetoFormationCustom struct {
		ApplicationName *string
		Tags            map[string]string
		Template        *Template
		StagingArea     *string
	}
	GadgetoFormationAdapter interface {
		MergeCommandTemplate(command *string, source *string, fileName *string) error
		SaveApplicationTemplate(fileName *string) error
	}

	Template struct {
		AWSTemplateFormatVersion string `yaml:"AWSTemplateFormatVersion"`
		//Transform                interface{}            `yaml:"Transform"`
		Description string                 `yaml:"Description"`
		Metadata    map[string]interface{} `yaml:"Metadata"`
		Parameters  map[string]interface{} `yaml:"Parameters"`
		Mappings    map[string]interface{} `yaml:"Mappings"`
		Conditions  map[string]interface{} `yaml:"Conditions"`
		Resources   map[string]interface{} `yaml:"Resources"`
		Outputs     map[string]interface{} `yaml:"Outputs"`
		//Globals                  map[string]interface{} `yaml:"Globals"`
	}
)

func createEmptyCloudformationTemplate() *Template {
	return &Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Parameters:               make(map[string]interface{}),
		Mappings:                 make(map[string]interface{}),
		Conditions:               make(map[string]interface{}),
		Resources:                make(map[string]interface{}),
		Outputs:                  make(map[string]interface{}),
		//Globals:                  make(map[string]interface{}),
	}
}

func (t *Template) applyToSelectiveResourceTypes(resourceTypePredicate util.ValuePredicate, applicator util.Applicator) error {
	for resourceKey, resourceRaw := range t.Resources {
		if resource, ok := resourceRaw.(map[interface{}]interface{}); ok {
			if resourceTypeRaw, found := resource["Type"]; found {
				if resourceType, ok := resourceTypeRaw.(string); ok {
					if resourceTypePredicate(resourceType) {
						err := applicator(resource)
						if err != nil {
							return err
						}
					}
				}
			}
		} else {
			return fmt.Errorf("could not process resource %s due to incompatible type %T", resourceKey, resourceRaw)
		}
	}
	return nil
}

func (t *Template) mergeElements(source map[interface{}]interface{}) error {
	var transformErr, descriptionErr error
	if _, found := source["Transform"]; found {
		transformErr = fmt.Errorf("cannot merge a template containing a transform statetment")
	}
	if _, found := source["Description"]; found {
		descriptionErr = fmt.Errorf("cannot merge a template containing a description")
	}
	paramErr := mapTopLevelProperty("Parameters", source, t.Parameters, util.OverwriteExisting)
	mappingErr := mapTopLevelProperty("Mappings", source, t.Mappings, util.OverwriteExisting)
	conditionErr := mapTopLevelProperty("Conditions", source, t.Conditions, util.OverwriteExisting)
	resourceErr := mapTopLevelProperty("Resources", source, t.Resources, util.Conflict)

	return errors.Join(transformErr, descriptionErr, paramErr, mappingErr, conditionErr, resourceErr)
}

func mapTopLevelProperty(key string, sourceMap map[interface{}]interface{}, target map[string]interface{}, mergeStrategy func(bool, interface{}, interface{}) (interface{}, error)) error {
	if sourceRaw, found := sourceMap[key]; found {
		if source, ok := sourceRaw.(map[interface{}]interface{}); ok {
			err := util.MergeMap(source, target, mergeStrategy)
			if err != nil {
				fmt.Printf("Target %v\n", target)
				return fmt.Errorf("could not process map[%s]: %w", key, err)
			}
			return nil
		}
		return fmt.Errorf("could not process map[%s] due to incompatible type %T", key, sourceRaw)
	}
	return nil
}

func NewGadgetoFormationAdapter(applicationName *string, tags map[string]string, stagingArea *string) (GadgetoFormationAdapter, error) {
	modVersion, err := readGoModule()
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	tags["org.gadget.source.module"] = modVersion
	tags["org.gadget.application.name"] = *applicationName

	return &GadgetoFormationCustom{
		ApplicationName: applicationName,
		Tags:            tags,
		Template:        createEmptyCloudformationTemplate(),
		StagingArea:     stagingArea,
	}, nil
}

func (g *GadgetoFormationCustom) MergeCommandTemplate(command *string, source *string, fileName *string) error {
	sourceMap, err := util.ReadYAMLFile(*fileName)
	if err != nil {
		return fmt.Errorf("could not read source file %s: %w", *fileName, err)
	}
	if err := g.Template.mergeElements(sourceMap.(map[interface{}]interface{})); err != nil {
		return err
	}
	commandTags := g.Tags
	commandTags["org.gadget.source.command.alias"] = *command
	commandTags["org.gadget.source.command.source"] = *source

	applicationTagsApplicator := util.GenerateTagApplicator(g.Tags)
	commandTagsApplicator := util.GenerateTagApplicator(commandTags)

	commandTagsErr := g.Template.applyToSelectiveResourceTypes(util.WhiteListCommandSpecificResourceTypes, commandTagsApplicator)
	applicationTagsErr := g.Template.applyToSelectiveResourceTypes(util.BlackListCommandSpecificResourceTypes, applicationTagsApplicator)

	return errors.Join(commandTagsErr, applicationTagsErr)
}

func (g *GadgetoFormationCustom) SaveApplicationTemplate(fileName *string) error {
	fullFileName := filepath.Join(*g.StagingArea, *fileName)
	return util.SaveYAMLFile(fullFileName, g.Template)
}

func readGoModule() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", err
	}
	version := file.Module.Mod
	return fmt.Sprintf("%s@%s", version.Path, version.Version), nil

}
