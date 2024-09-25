package v2

import (
	"encoding/json"
	"errors"
	v0 "github.com/ashbeelghouri/jsonschematics/data/v0"
	"github.com/ashbeelghouri/jsonschematics/operators"
	"github.com/ashbeelghouri/jsonschematics/utils"
	"github.com/ashbeelghouri/jsonschematics/validators"
	"os"
)

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

type Schema struct {
	Version string                 `json:"version"`
	Fields  []Field                `json:"fields"`
	DB      map[string]interface{} `json:"DB"`
}

type Field struct {
	DependsOn             []string               `json:"depends_on"`
	DisplayName           string                 `json:"display_name"`
	Name                  string                 `json:"name"`
	TargetKey             string                 `json:"target_key"`
	AddToDB               bool                   `json:"add_to_db"`
	Type                  string                 `json:"type"`
	IsRequired            bool                   `json:"required"`
	Description           string                 `json:"description"`
	Validators            []Component            `json:"validators"`
	Operators             []Component            `json:"operators"`
	Conditions            []Condition            `json:"conditions"`
	L10n                  map[string]interface{} `json:"l10n"`
	AdditionalInformation map[string]interface{} `json:"additional_information"`
}

type Condition struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

type ComponentLocale struct {
	Name  map[string]interface{} `json:"name"`
	Error map[string]interface{} `json:"error"`
}

type Component struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
	Error      string                 `json:"error"`
	L10n       ComponentLocale        `json:"l10n"`
}

func LoadJsonSchemaFile(path string) (*v0.Schematics, error) {
	var s Schematics
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schema Schema
	err = json.Unmarshal(content, &schema)
	if err != nil {
		return nil, err
	}
	s.Schema = schema
	baseSchematics := transformSchematics(s)
	if baseSchematics != nil {
		return baseSchematics, nil
	} else {
		return nil, errors.New("could not load the base schema")
	}
}

func LoadMap(schemaMap interface{}) (*v0.Schematics, error) {
	var s Schematics
	jsonBytes, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, err
	}
	var schema Schema
	err = json.Unmarshal(jsonBytes, &schema)
	if err != nil {
		return nil, err
	}
	s.Schema = schema
	return transformSchematics(s), nil
}

func transformSchematics(s Schematics) *v0.Schematics {
	var baseSchematics v0.Schematics
	if s.Logging.PrintDebugLogs {
		baseSchematics.Logging.PrintDebugLogs = true
	}
	if s.Logging.PrintErrorLogs {
		baseSchematics.Logging.PrintErrorLogs = true
	}

	baseSchematics.ArrayIdKey = s.ArrayIdKey
	baseSchematics.Separator = s.Separator
	baseSchematics.Validators.BasicValidators()
	baseSchematics.Operators.LoadBasicOperations()
	baseSchematics.Schema = *transformSchema(s.Schema)
	if s.DB != nil {
		baseSchematics.Schema.DB = utils.CombineTwoMaps(baseSchematics.Schema.DB, s.DB)
	}
	return &baseSchematics
}

func transformSchema(schema Schema) *v0.Schema {
	var baseSchema v0.Schema
	baseSchema.Version = schema.Version
	baseSchema.DB = schema.DB
	baseSchema.Fields = make(map[v0.TargetKey]v0.Field)

	for _, field := range schema.Fields {
		baseSchema.Fields[v0.TargetKey(field.TargetKey)] = v0.Field{
			DependsOn:             field.DependsOn,
			Name:                  field.Name,
			AddToDB:               field.AddToDB,
			Type:                  field.Name,
			IsRequired:            field.IsRequired,
			Description:           field.Description,
			Validators:            transformComponents(field.Validators),
			Operators:             transformComponents(field.Operators),
			Conditions:            transformConditions(field.Conditions),
			L10n:                  field.L10n,
			AdditionalInformation: field.AdditionalInformation,
		}
	}
	return &baseSchema
}

func CreateConstantLocale(locale ComponentLocale) v0.ConstantL10n {
	var c = v0.ConstantL10n{
		Name:  locale.Name,
		Error: locale.Error,
	}
	return c
}

func transformComponents(comp []Component) map[string]v0.Constant {
	con := make(map[string]v0.Constant)
	for _, c := range comp {
		con[c.Name] = v0.Constant{
			Attributes: c.Attributes,
			Error:      c.Error,
			L10n:       CreateConstantLocale(c.L10n),
		}
	}
	return con
}

func transformConditions(cond []Condition) map[string]v0.Condition {
	con := make(map[string]v0.Condition)
	for _, c := range cond {
		con[c.Name] = v0.Condition{
			Attributes: c.Attributes,
		}
	}
	return con
}
