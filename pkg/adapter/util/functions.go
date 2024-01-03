package util

import "fmt"

func KeepExisting(existing bool, source interface{}, target interface{}) (interface{}, error) {
	return target, nil
}

func OverwriteExisting(existing bool, source interface{}, target interface{}) (interface{}, error) {
	return source, nil
}

func Conflict(existing bool, source interface{}, target interface{}) (interface{}, error) {
	return nil, fmt.Errorf("key already exists")
}

func MergeMap(source map[interface{}]interface{}, target map[string]interface{}, mergeStrategy func(bool, interface{}, interface{}) (interface{}, error)) error {
	if target == nil {
		return fmt.Errorf("no target available to merge into")
	}
	for keyRaw, sourceValue := range source {
		key, ok := keyRaw.(string)
		if !ok {
			return fmt.Errorf("could not process map key %v due to incompatible type %T, needed %T", keyRaw, keyRaw, key)
		}
		if targetValue, found := target[key]; found {
			mergedValue, err := mergeStrategy(true, sourceValue, targetValue)
			if err != nil {
				return err
			}
			target[key] = mergedValue
		} else {
			target[key] = sourceValue
		}
	}
	return nil
}

type Applicator func(map[interface{}]interface{}) error

type ValuePredicate func(value string) bool

var CommandSpecificResourceTypes = []string{
	"AWS::Lambda::Function",
	"AWS::IAM::Role",
}

var WhiteListCommandSpecificResourceTypes = WhiteListPredicate(CommandSpecificResourceTypes)
var BlackListCommandSpecificResourceTypes = BlackListPredicate(CommandSpecificResourceTypes)

func WhiteListPredicate(whitelist []string) ValuePredicate {
	return func(value string) bool {
		for _, allowed := range whitelist {
			if value == allowed {
				return true
			}
		}
		return false
	}
}

func BlackListPredicate(blacklist []string) ValuePredicate {
	return func(value string) bool {
		for _, allowed := range blacklist {
			if value == allowed {
				return false
			}
		}
		return true
	}
}

func GenerateTagApplicator(tags map[string]string) Applicator {
	return func(resource map[interface{}]interface{}) error {
		if _, found := resource["Properties"]; found {
			properties := resource["Properties"].(map[interface{}]interface{})
			if rawTags, found := properties["Tags"]; found {
				if destTags, ok := rawTags.([]interface{}); ok {
					for key, value := range tags {
						destTags = append(destTags, map[interface{}]interface{}{
							"Key":   key,
							"Value": value,
						})
					}
					properties["Tags"] = destTags
				} else if destTags, ok := rawTags.(map[interface{}]interface{}); ok {
					for key, value := range tags {
						destTags[key] = value
					}
				} else {
					return fmt.Errorf("could not process tags due to incompatible type %T", properties["Tags"])
				}
			}
		}
		return nil
	}
}
