package migrations

import "github.com/pocketbase/pocketbase/models/schema"

var sites = schema.NewSchema(
	&schema.SchemaField{
		Name:     "site",
		Type:     schema.FieldTypeText,
		Unique:   true,
		Required: true,
	},
)
