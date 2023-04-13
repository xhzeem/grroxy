package migrations

import "github.com/pocketbase/pocketbase/models/schema"

var settings = schema.NewSchema(
	&schema.SchemaField{
		Name:     "option",
		Type:     schema.FieldTypeText,
		Unique:   true,
		Required: true,
	},
	&schema.SchemaField{
		Name: "value",
		Type: schema.FieldTypeText,
	},
)
