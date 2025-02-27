// Initial setup to start the db file with pre build tables and user
package migrations

import (
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

type setting struct {
	ID    string
	Name  string
	Value string
}

func init() {
	m.Register(func(db dbx.Builder) error {

		// you can also access the Dao helpers
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			log.Println(err)
		}

		//Delete users
		if err := dao.DeleteCollection(collection); err != nil {
			log.Println(err)
		}

		// create admin
		admin := &models.Admin{
			Email:        "new@example.com",
			PasswordHash: "$2a$13$1EIwr9jv9bJJxfIUd.EtrOGfXCWAm.NuaFt6ZG6OlWmHSUE1Wwdi.",
		}

		if err := dao.SaveAdmin(admin); err != nil {
			log.Println(err)
		}

		return nil
	}, func(db dbx.Builder) error {
		// revert changes...

		return nil
	})
}
