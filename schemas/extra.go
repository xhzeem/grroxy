package schemas

import "github.com/pocketbase/pocketbase/models/schema"

var Extra = schema.NewSchema(
	&schema.SchemaField{
		Name: "label",
		Type: schema.FieldTypeJson,
	},
	&schema.SchemaField{
		Name: "note",
		Type: schema.FieldTypeEditor,
	},
)
