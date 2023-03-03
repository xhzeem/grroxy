package main

import (
	"log"
	"net/http"

	// "github.com/pocketbase/dbx"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"

	// "github.com/pocketbase/pocketbase/tools/list"
	grroxyTypes "github.com/glitchedgitz/grroxy/pkg/types"

	"github.com/pocketbase/pocketbase/tools/list"
	"github.com/pocketbase/pocketbase/tools/types"
)

type databaseAPI struct {
	app *pocketbase.PocketBase
}

func (pocketbaseDB *databaseAPI) getSitemap(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/sitemap",
		Handler: func(c echo.Context) error {

			var data Data
			if err := c.Bind(&data); err != nil {
				return err
			}

			log.Println("Request: ", data)

			var result []grroxyTypes.UserData

			err := pocketbaseDB.app.DB()

			log.Println("Result: ", result)

			if err != nil {
				apis.NewBadRequestError("Failed to fetch warehouse items", err)
			}

			return c.JSON(http.StatusOK, result)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.app),
		},
	})

	return nil
}

func (pocketbaseDB *databaseAPI) newSitemap(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/create",
		Handler: func(c echo.Context) error {
			name := c.QueryParam("sitemap")
			log.Println("New Collection: ", name)

			collection := &models.Collection{
				Name:       name,
				Type:       models.CollectionTypeBase,
				ListRule:   nil,
				ViewRule:   types.Pointer(""),
				CreateRule: types.Pointer(""),
				UpdateRule: types.Pointer(""),
				DeleteRule: nil,
				Schema: schema.NewSchema(
					&schema.SchemaField{
						Name:     "path",
						Type:     schema.FieldTypeText,
						Required: true,
						Unique:   true,
					}, &schema.SchemaField{
						Name:     "type",
						Type:     schema.FieldTypeText,
						Required: true,
					},
					&schema.SchemaField{
						Name:     "mainId",
						Type:     schema.FieldTypeText,
						Required: true,
					},
				),
			}

			if err := pocketbaseDB.app.Dao().SaveCollection(collection); err != nil {
				apis.NewBadRequestError("Failed to create sitemap table", err)
			}

			return c.String(http.StatusOK, "Created")
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.app),
		},
	})
	return nil
}

func (pocketbaseDB *databaseAPI) getData(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/data",
		Handler: func(c echo.Context) error {

			var data Data
			if err := c.Bind(&data); err != nil {
				return err
			}

			ids := data.Ids
			log.Println("Request: ", data)

			var result []grroxyTypes.UserData

			err := pocketbaseDB.app.Dao().DB().
				Select("*").
				From("data").
				Where(dbx.In(
					"id",
					list.ToInterfaceSlice(ids)...,
				)).
				OrderBy("created desc").
				Limit(data.PerPage).Offset(data.Page * data.PerPage).
				All(&result)

			log.Println("Result: ", result)

			if err != nil {
				apis.NewBadRequestError("Failed to fetch warehouse items", err)
			}

			return c.JSON(http.StatusOK, result)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.app),
		},
	})

	return nil
}

type Data struct {
	Ids     []string `json:"ids"`
	Page    int64    `json:""`
	PerPage int64    `json:""`
}

func main() {
	// Create an instance of the app structure
	pocketbaseDB := databaseAPI{
		app: pocketbase.New(),
	}
	pocketbaseDB.app.OnBeforeServe().Add(pocketbaseDB.getData)
	pocketbaseDB.app.OnBeforeServe().Add(pocketbaseDB.newSitemap)
	pocketbaseDB.app.OnBeforeServe().Add(pocketbaseDB.getSitemap)

	if err := pocketbaseDB.app.Start(); err != nil {
		log.Fatal(err)
	}
}
