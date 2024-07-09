package v0

import (
	"encoding/json"
	"fmt"
	"github.com/DScale-io/jsonschematics/errorHandler"
	"github.com/DScale-io/jsonschematics/operators"
	"github.com/DScale-io/jsonschematics/utils"
	"github.com/DScale-io/jsonschematics/validators"
	"log"
	"os"
	"strings"
)

type TargetKey string

type Schematics struct {
	Schema     Schema
	Validators validators.Validators
	Operators  operators.Operators
	Separator  string
	ArrayIdKey string
	Locale     string
	DB         map[string]interface{}
	Logging    utils.Logger
}

// add this DB to the attributes as SCHEMA_GLOBAL_DB

type Schema struct {
	Version string                 `json:"version"`
	Fields  map[TargetKey]Field    `json:"fields"`
	DB      map[string]interface{} `json:"DB"`
}

type Field struct {
	DependsOn             []string               `json:"depends_on"`
	DisplayName           string                 `json:"display_name"`
	Name                  string                 `json:"name"`
	Type                  string                 `json:"type"`
	IsRequired            bool                   `json:"required"`
	AddToDB               bool                   `json:"add_to_db"`
	Description           string                 `json:"description"`
	Validators            map[string]Constant    `json:"validators"`
	Operators             map[string]Constant    `json:"operators"`
	L10n                  map[string]interface{} `json:"l10n"`
	AdditionalInformation map[string]interface{} `json:"additional_information"`
	logging               utils.Logger
}

type ConstantL10n struct {
	Name  map[string]interface{} `json:"name"`
	Error map[string]interface{} `json:"error"`
}

type Constant struct {
	Attributes map[string]interface{} `json:"attributes"`
	Error      string                 `json:"error"`
	L10n       ConstantL10n           `json:"l10n"`
}

func (s *Schematics) Configs() {
	if s.Logging.PrintDebugLogs {
		log.Println("debugger is on")
	}
	if s.Logging.PrintErrorLogs {
		log.Println("error logging is on")
	}
	s.Validators.Logger = s.Logging
	s.Operators.Logger = s.Logging
}

func (s *Schematics) LoadJsonSchemaFile(path string) error {
	s.Configs()
	content, err := os.ReadFile(path)
	if err != nil {
		s.Logging.ERROR("Failed to load schema file", err)
		return err
	}
	var schema Schema
	err = json.Unmarshal(content, &schema)
	if err != nil {
		s.Logging.ERROR("Failed to unmarshall schema file", err)
		return err
	}
	s.Logging.DEBUG("Schema Loaded From File: ", schema)
	s.Schema = schema
	s.Validators.BasicValidators()
	s.Operators.LoadBasicOperations()
	if s.Separator == "" {
		s.Separator = "."
	}
	if s.Locale == "" {
		s.Locale = "en"
	}
	return nil
}

func (s *Schematics) LoadMap(schemaMap interface{}) error {
	JSON, err := json.Marshal(schemaMap)
	if err != nil {
		s.Logging.ERROR("Schema should be valid json map[string]interface", err)
		return err
	}
	var schema Schema
	err = json.Unmarshal(JSON, &schema)
	if err != nil {
		s.Logging.ERROR("Invalid Schema", err)
		return err
	}
	s.Logging.DEBUG("Schema Loaded From MAP: ", schema)
	s.Schema = schema
	s.Validators.BasicValidators()
	s.Operators.LoadBasicOperations()
	if s.Separator == "" {
		s.Separator = "."
	}
	if s.Locale == "" {
		s.Locale = "en"
	}
	return nil
}

// if validators >>> if passed then do *

func (f *Field) Validate(value interface{}, allValidators map[string]validators.Validator, id *string, db map[string]interface{}) *errorHandler.Error {
	var err errorHandler.Error
	err.Value = value
	err.ID = id
	err.Validator = "unknown"
	if f.Validators == nil {
		err.AddMessage("en", "no validators defined")
		return &err
	}
	for name, constants := range f.Validators {
		err.Validator = name
		f.logging.DEBUG("Validator", name, constants)
		if name == "" {
			f.logging.DEBUG("Name of the validator is not given: ", name)
			err.Validator = name
			err.AddMessage("en", "no validator name given")
			return &err
		}
		if f.IsRequired && value == nil {
			err.Validator = "Required"
			err.AddMessage("en", "this is a required field")
			f.logging.DEBUG("Field is required but value is null")
			return &err
		}

		if utils.StringInStrings(strings.ToUpper(name), utils.ExcludedValidators) {
			continue
		}

		var fn validators.Validator
		fn, exists := allValidators[name]
		f.logging.DEBUG("function exists? ", exists)
		if !exists {
			f.logging.ERROR("function not found", name)
			err.AddMessage("en", "validator not registered")
			return &err
		}

		if constants.Attributes == nil {
			constants.Attributes = make(map[string]interface{})
		}
		constants.Attributes["DB"] = db
		fnError := fn(value, constants.Attributes)
		f.logging.DEBUG("fnError: ", fnError)
		if fnError != nil {
			err.AddMessage("en", fnError.Error())
			if constants.Error != "" {
				f.logging.DEBUG("Custom Error is Defined", constants.Error)
				err.AddMessage("en", constants.Error)
			}

			if f.L10n != nil {
				for locale, msg := range constants.L10n.Error {
					if msg != nil {
						f.logging.DEBUG("Error L10n: ", locale, msg)
						err.AddMessage(locale, msg.(string))
					}
				}

				for local, v := range constants.L10n.Name {
					if v != nil {
						f.logging.DEBUG("Validator L10n: ", local, v)
						err.AddL10n(name, local, v.(string))
					}
				}
			}
			return &err
		}
	}
	return nil
}

func (s *Schematics) makeFlat(data map[string]interface{}) *map[string]interface{} {
	var dMap utils.DataMap
	dMap.FlattenTheMap(data, "", s.Separator)
	return &dMap.Data
}

func (s *Schematics) deflate(data map[string]interface{}) map[string]interface{} {
	return utils.DeflateMap(data, s.Separator)
}

func (s *Schematics) Validate(jsonData interface{}) *errorHandler.Errors {
	var baseError errorHandler.Error
	var errs errorHandler.Errors
	baseError.Validator = "validate-object"
	if s == nil {
		baseError.AddMessage("en", "schema not loaded")
		errs.AddError("whole-data", baseError)
		return &errs
	}

	dataBytes, err := json.Marshal(jsonData)
	if err != nil {
		baseError.AddMessage("en", "data is not valid json")
		errs.AddError("whole-data", baseError)
		return &errs
	}

	var obj map[string]interface{}
	var arr []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &obj); err == nil {
		return s.ValidateObject(&obj, nil)
	} else if err := json.Unmarshal(dataBytes, &arr); err == nil {
		return s.ValidateArray(arr)
	} else {
		baseError.AddMessage("en", "invalid format provided for the data, can only be map[string]interface or []map[string]interface")
		errs.AddError("whole-data", baseError)
		return &errs
	}
}

func (s *Schematics) ValidateObject(jsonData *map[string]interface{}, id *string) *errorHandler.Errors {
	s.Logging.DEBUG("validating the object")
	var errorMessages errorHandler.Errors
	var baseError errorHandler.Error
	flatData := *s.makeFlat(*jsonData)
	s.Logging.DEBUG("here after flat data --> ", flatData)
	uniqueID := ""

	if id != nil {
		uniqueID = *id
	}
	s.Logging.DEBUG("after unique id")

	db := s.Schema.GetDB(flatData)

	var missingFromDependants []string
	for target, field := range s.Schema.Fields {
		field.logging = s.Logging
		baseError.Validator = "is-required"
		matchingKeys := utils.FindMatchingKeys(flatData, string(target))
		s.Logging.DEBUG("matching keys --> ", matchingKeys)
		if len(matchingKeys) == 0 {
			if field.IsRequired {
				baseError.AddMessage("en", "this field is required")
				errorMessages.AddError(string(target), baseError)
			}
			continue
		}
		s.Logging.DEBUG("after is required --> ", matchingKeys)
		//	check for dependencies
		if len(field.DependsOn) > 0 {
			missing := false
			for _, d := range field.DependsOn {
				matchDependsOn := utils.FindMatchingKeys(flatData, d)
				if !(utils.StringInStrings(string(target), missingFromDependants) == false && len(matchDependsOn) > 0) {
					s.Logging.DEBUG("matched depends on", matchDependsOn)
					baseError.Validator = "depends-on"
					baseError.AddMessage("en", "this field depends on other values which do not exists")
					errorMessages.AddError(string(target), baseError)
					missingFromDependants = append(missingFromDependants, string(target))
					missing = true
					break
				}
			}
			if missing {
				continue
			}
		}

		for key, value := range matchingKeys {
			validationError := field.Validate(value, s.Validators.ValidationFns, &uniqueID, db)
			s.Logging.DEBUG(validationError)
			if validationError != nil {
				errorMessages.AddError(key, *validationError)
			}
		}

	}

	if errorMessages.HasErrors() {
		return &errorMessages
	}
	return nil
}

// Corrected and completed GetDB function
func (s *Schema) GetDB(flatData map[string]interface{}) map[string]interface{} {
	db := s.DB
	for target, field := range s.Fields {
		if field.AddToDB {
			matchingKeys := utils.FindMatchingKeys(flatData, string(target))
			if len(matchingKeys) < 2 {
				mappedKey := utils.GetFirstFromMap(matchingKeys)
				if mappedKey != nil {
					db[string(target)] = mappedKey
				}
			} else if len(matchingKeys) > 0 {
				var values []interface{}
				for _, match := range matchingKeys {
					values = append(values, match)
				}
				db[string(target)] = values
			}
		}
	}
	return db
}

func (s *Schematics) ValidateArray(jsonData []map[string]interface{}) *errorHandler.Errors {
	s.Logging.DEBUG("validating the array")
	var errs errorHandler.Errors
	i := 0
	for _, d := range jsonData {
		var errorMessages *errorHandler.Errors
		var dMap utils.DataMap
		dMap.FlattenTheMap(d, "", s.Separator)
		arrayId, exists := dMap.Data[s.ArrayIdKey]
		if !exists {
			arrayId = fmt.Sprintf("row-%d", i)
			exists = true
		}

		id := arrayId.(string)
		errorMessages = s.ValidateObject(&d, &id)
		if errorMessages.HasErrors() {
			s.Logging.ERROR("has errors", errorMessages.GetStrings("en", "%data\n"))
			errs.MergeErrors(errorMessages)
		}
		i = i + 1
	}

	if errs.HasErrors() {
		return &errs
	}
	return nil
}

// operators

func (f *Field) Operate(value interface{}, allOperations map[string]operators.Op) interface{} {
	for operationName, operationConstants := range f.Operators {
		customValidator, exists := allOperations[operationName]
		if !exists {
			f.logging.ERROR("This operation does not exists in basic or custom operators", operationName)
			return nil
		}
		result := customValidator(value, operationConstants.Attributes)
		if result != nil {
			value = result
		}
	}
	return value
}

func (s *Schematics) Operate(data interface{}) (interface{}, *errorHandler.Errors) {
	var errorMessages errorHandler.Errors
	var baseError errorHandler.Error
	baseError.Validator = "operate-on-schema"
	bytes, err := json.Marshal(data)
	if err != nil {
		s.Logging.ERROR("[operate] error converting the data into bytes", err)
		baseError.AddMessage("en", "data is not valid json")
		errorMessages.AddError("whole-data", baseError)
		return nil, &errorMessages
	}

	dataType, item := utils.IsValidJson(bytes)
	if item == nil {
		s.Logging.ERROR("[operate] error occurred when checking if this data is an array or object")
		baseError.AddMessage("en", "can not convert the data into json")
		errorMessages.AddError("whole-data", baseError)
		return nil, &errorMessages
	}

	if dataType == "object" {
		obj := item.(map[string]interface{})
		results := s.OperateOnObject(obj)
		if results != nil {
			return results, nil
		} else {
			baseError.AddMessage("en", "operation on object unsuccessful")
			errorMessages.AddError("whole-data", baseError)
			return nil, &errorMessages
		}
	} else if dataType == "array" {
		arr := item.([]map[string]interface{})
		results := s.OperateOnArray(arr)
		if results != nil && len(*results) > 0 {
			return results, nil
		} else {
			baseError.AddMessage("en", "operation on array unsuccessful")
			errorMessages.AddError("whole-data", baseError)
			return nil, &errorMessages
		}
	}

	return data, nil
}

func (s *Schematics) OperateOnObject(data map[string]interface{}) *map[string]interface{} {
	data = *s.makeFlat(data)
	for target, field := range s.Schema.Fields {
		matchingKeys := utils.FindMatchingKeys(data, string(target))
		for key, value := range matchingKeys {
			data[key] = field.Operate(value, s.Operators.OpFunctions)
		}
	}
	d := s.deflate(data)
	return &d
}

func (s *Schematics) OperateOnArray(data []map[string]interface{}) *[]map[string]interface{} {
	var obj []map[string]interface{}
	for _, d := range data {
		results := s.OperateOnObject(d)
		obj = append(obj, *results)
	}
	if len(obj) > 0 {
		return &obj
	}
	return nil
}

// General

func (s *Schematics) MergeFields(sc2 *Schematics) *Schematics {
	for target, field := range sc2.Schema.Fields {
		if s.Schema.Fields[target].Type == "" {
			s.Schema.Fields[target] = field
		}
	}
	return s
}
