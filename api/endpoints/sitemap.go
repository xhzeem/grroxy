package endpoints

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/schemas"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (pocketbaseDB *DatabaseAPI) SitemapNew(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/sitemap/new",
		Handler: func(c echo.Context) error {

			var data types.SitemapGet
			if err := c.Bind(&data); err != nil {
				return err
			}

			fmt.Print("SitemapNew: ", data)
			collection := base.ParseDatabaseName(data.Host)

			err := pocketbaseDB.CreateCollection(collection, schemas.Sitemap)

			// Checking error if it is collection already exists
			// This is the error "constraint failed: UNIQUE constraint failed: collections.name (2067)"

			if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
				log.Println("collection already exists: ", collection)
			}

			// Inserting data
			result, err := pocketbaseDB.App.Dao().DB().Insert(collection, dbx.Params{
				"id":       data.MainID,
				"path":     data.Path,
				"query":    data.Query,
				"fragment": data.Fragment,
				"type":     data.Type,
				"main_id":  data.MainID,
			}).Execute()

			log.Println("Executed: ", result)

			if err != nil {
				// return nil
				fmt.Println("Error: ", err)
				// apis.NewBadRequestError("Failed to create collection", err)
			}

			return c.String(http.StatusOK, "Created")
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})
	return nil
}
