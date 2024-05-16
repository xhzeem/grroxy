package templates

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/glitchedgitz/filters"
	"github.com/glitchedgitz/grroxy-db/base"
	"gopkg.in/yaml.v2"
)

type Action struct {
	Data       map[string]any `yaml:"data"`
	ActionName string         `yaml:"action_name"`
}

type Actions struct {
	Filter  string   `yaml:"filter"`
	Actions []Action `yaml:"actions"`
}

type Info struct {
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
	Author      string `yaml:"author,omitempty"`
}

type Template struct {
	Id   string `yaml:"id"`
	Info Info   `yaml:"info"`

	// Mode?: By default it's 'all',
	//        Use 'any' to stop after one match
	Mode string `yaml:"mode,omitempty"`

	// Hooks: Which templates one should run
	On map[string][]string `yaml:"on,omitempty"`

	// Default?: actions to perform when nothing match
	Default []Action `yaml:"default,omitempty"`

	// ActionsList: List of actions to check
	ActionsList []Actions `yaml:"actionslist"`
}

type Templates struct {
	List []*Template
}

func Setup() *Templates {
	var t Templates

	files, err := os.ReadDir(`D:\sdks\go\src\github.com\glitchedgitz\grroxy-db\grroxy-templates`)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("[Template.Setup]")

	for _, file := range files {
		l := Read(`D:\sdks\go\src\github.com\glitchedgitz\grroxy-db\grroxy-templates\` + file.Name())
		log.Printf("Template:%v Scan:%v\n", file.Name(), len(l.ActionsList))
		log.Printf("Template:%v Mode:%v\n", file.Name(), l.Mode)
		t.List = append(t.List, l)
	}

	return &t
}

func Read(filePath string) *Template {

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	// Unmarshal yaml
	var y Template

	err = yaml.Unmarshal(yamlFile, &y)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return &y
}

func getParsedValue(data map[string]any, action Action) Action {
	a := Action{
		Data:       make(map[string]any),
		ActionName: action.ActionName,
	}
	for key, str := range action.Data {
		if strVal, ok := str.(string); ok {
			parsedValue := ParseVariables(&data, strVal)
			// if err != nil {
			// 	a.Data[key] = "error"
			// } else {
			a.Data[key] = parsedValue
			// }
		} else {
			a.Data[key] = str
		}
	}
	return a
}

func (t *Templates) Run(data map[string]any, hook string) ([]Action, error) {

	results := []Action{}

	log.Println("[Templates.Run] data", data)

	hooks := strings.Split(hook, ":")

	for _, template := range t.List {

		var actions []Action

		if len(template.ActionsList) == 0 {
			log.Println("No actions found") // Fail if no actions are found
			continue
		}

		if values, found := template.On[hooks[0]]; !found {
			continue
		} else {
			foundAny := false
			for _, hook := range hooks[1:] {
				if base.ArrayContains(values, hook) {
					foundAny = true
				}
			}
			if !foundAny {
				continue
			}
		}

		log.Printf("[Templates.Run] Running template: %s", template.Info.Title)

		for _, job := range template.ActionsList {
			check, err := filters.Filter(data, job.Filter)
			if err != nil {
				log.Printf("[Templates.Run] Filter parsing: %v", job.Filter)
				break
			}

			if check {
				log.Println("[template.Run] Found with:", job.Filter)

				for _, action := range job.Actions {
					actions = append(actions, getParsedValue(data, action))
				}

				if template.Mode == "any" {
					if len(actions) > 0 {
						break
					}
				}
			}
		}

		if len(actions) == 0 {
			if len(template.Default) > 0 {
				log.Println("[template.Run] Using default for", data)

				for _, _action := range template.Default {
					log.Println("Before even sending: ", _action)
					results = append(results, getParsedValue(data, _action))
				}
			}
		}

		results = append(results, actions...)
	}

	return results, nil
}

func ParseVariables(d *map[string]any, value string) string {

	log.Println("[ParseVariables] Using data ", *d)

	re := regexp.MustCompile(`{{(.*?)}}`)

	// Find all matches
	matches := re.FindAllStringSubmatch(value, -1)

	// Extract the captured groups
	for _, match := range matches {
		if len(match) > 1 {
			field := match[1]
			if strings.Contains(field, ".") {
				var left string
				var dataset map[string]any

				k := strings.Split(field, ".")
				klen := len(k)

				tmpdata := *d

				for j, key := range k {
					if j < klen-1 {
						left = k[j+1]
						if b, found := tmpdata[key]; found {
							switch b := b.(type) {
							case map[string]any:
								tmpdata = b
							default:
								break
							}
						}
					}
				}

				dataset = tmpdata
				extracted_value, err := extractValueIfKeyExists(&dataset, left)
				if err != nil {
					return "error"
				}
				value = strings.ReplaceAll(value, match[0], extracted_value)
			} else {
				extracted_value, err := extractValueIfKeyExists(d, field)
				if err != nil {
					return "error"
				}
				value = strings.ReplaceAll(value, match[0], extracted_value)
			}
		}
	}

	return value
}

func extractValueIfKeyExists(d *map[string]any, key string) (string, error) {
	if b, found := (*d)[key]; found {
		switch b := b.(type) {
		case string:
			return b, nil
		default:
			return "", fmt.Errorf("not string")
		}
	}
	return "", fmt.Errorf("key '%v' not found", key)
}
