package migrations

import (
	"log"

	"github.com/glitchedgitz/grroxy-db/schemas"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	pbTypes "github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// add up queries...
		collection := &models.Collection{
			Name:       "_processes",
			Type:       models.CollectionTypeBase,
			ListRule:   pbTypes.Pointer(""),
			ViewRule:   pbTypes.Pointer(""),
			CreateRule: pbTypes.Pointer(""),
			UpdateRule: pbTypes.Pointer(""),
			DeleteRule: nil,
			Schema:     schemas.PROCESSES,
		}

		collection.SetId("_processes")

		if err := dao.SaveCollection(collection); err != nil {
			log.Println("[migration][_processes] Error: ", err)
		}

		log.Println("[migration][_processes] Creating collection: ", "_ui")
		return nil
	}, func(db dbx.Builder) error {
		// add down queries...

		return nil
	})
}
