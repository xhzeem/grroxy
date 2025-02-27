package schemas

import "github.com/pocketbase/pocketbase/models/schema"

var Projects = schema.NewSchema(
	&schema.SchemaField{
		Name: "name",
		Type: schema.FieldTypeText,
	},
	&schema.SchemaField{
		Name: "path",
		Type: schema.FieldTypeText,
	},
	&schema.SchemaField{
		Name: "ip",
		Type: schema.FieldTypeText,
	},
	&schema.SchemaField{
		Name: "state",
		Type: schema.FieldTypeText,
	},
	&schema.SchemaField{
		Name: "version",
		Type: schema.FieldTypeText,
	},
)
