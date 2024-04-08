package api

import (
	"log"

	"github.com/pocketbase/pocketbase/models"
)

func (backend *Backend) SaveRecordToCollection(collectionName string, data map[string]any) {
	collection, err := backend.App.Dao().FindCollectionByNameOrId(collectionName)
	if err != nil {
		log.Println("Error: ", err)
	}

	record := models.NewRecord(collection)

	for key, value := range data {
		record.Set(key, value)
	}

	err = backend.App.Dao().Save(record)

	if err != nil {
		log.Println("Error: ", err)
	}
}
