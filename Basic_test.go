package jsonschematics

import (
	v2 "github.com/DScale-io/jsonschematics/data/v2"
	"github.com/DScale-io/jsonschematics/utils"
	"log"
	"os"
	"testing"
)

func customFunc(i interface{}, attr map[string]interface{}) error {
	log.Println("--------->>> ========== >>>>> attributes are: ", attr)
	return nil
}

func TestV2Validate(t *testing.T) {
	schematics, err := v2.LoadJsonSchemaFile("test-data/schema/direct/v2/example-1.json")
	if err != nil {
		t.Error(err)
	}
	schematics.Logging.PrintDebugLogs = true
	schematics.Logging.PrintErrorLogs = true
	schematics.Validators.RegisterValidator("NewFun", NewFun)
	schematics.Validators.RegisterValidator("customFunc", customFunc)
	content, err := os.ReadFile("test-data/data/direct/example.json")
	if err != nil {
		t.Error(err)
	}
	jsonData, err := utils.BytesToMap(content)
	if err != nil {
		t.Error(err)
	}
	errs := schematics.Validate(jsonData)
	log.Println(errs.GetStrings("ar", "%message\n"))
}
func NewFun(i interface{}, attr map[string]interface{}) error {
	log.Println(i)
	log.Println(attr)
	return nil
}
