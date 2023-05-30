package endpoints

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var mapTypeToSQLType = map[string]string{
	"string":  "TEXT",
	"number":  "INTEGER",
	"boolean": "BOOLEAN",
	"object":  "JSON",
	"array":   "JSON",
	"null":    "JSON",
}

// OrderBy: Create orderby column, -column, column.key.subkey, -column.key.subkey
func orderBy(sort string, coltype string) string {

	direction := "asc"

	if sort == "" || coltype == "" {
		return "created desc"
	}

	if strings.HasPrefix(sort, "-") {
		direction = "desc"
		sort = sort[1:]
	}

	if strings.Contains(sort, ".") {
		t := strings.Split(sort, ".")
		column := t[0]
		extract := strings.Join(t[1:], ".")

		s := fmt.Sprintf("CAST(json_extract(%s, '$.%s') AS %s) %s", column, extract, mapTypeToSQLType[coltype], direction)
		log.Println("OrderBy: ", s)
		// SQL: json_extract(YourJsonColumn, '$.status_code')
		return s
	}

	return sort + " " + direction
}

func (pocketbaseDB *DatabaseAPI) fetchAllRows(data types.GetData) []types.UserData2 {
	var results []types.UserData2

	err := pocketbaseDB.App.Dao().DB().
		NewQuery(fmt.Sprintf(
			"SELECT * FROM data ORDER BY %s LIMIT %d OFFSET %d",
			orderBy(data.Sort, data.ColType),
			data.PerPage,
			data.Page*data.PerPage,
		)).All(&results)

	if err != nil {
		apis.NewBadRequestError("Failed to fetch warehouse items", err)
	}

	return results
}

func (pocketbaseDB *DatabaseAPI) fetchSitemapRows(data types.GetData) []types.UserData2 {
	// its a sitemap
	db := base.ParseDatabaseName(data.Collection)
	type MainIDPath struct {
		MainID string `db:"mainID"`
		Path   string `db:"path"`
	}

	var err error
	var results []types.UserData2

	// var mainIDPathResults []MainIDPath
	// if data.Path == "" || data.Path == "/" {
	// 	err = pocketbaseDB.App.Dao().DB().Select("mainID", "path").From(db).All(&mainIDPathResults)
	// } else {
	// 	regexQuery := data.Path + `/%`
	// 	err = pocketbaseDB.App.Dao().DB().NewQuery("SELECT mainID,path FROM " + db + " WHERE path LIKE '" + regexQuery + "'").All(&mainIDPathResults)
	// }

	// log.Println("[SitemapRows] mainIDPathResults: ", mainIDPathResults)
	// if err != nil {
	// 	apis.NewBadRequestError("Failed to fetch warehouse items", err)
	// }

	// uniqueFolders := make(map[string]bool)
	// folders := []string{}
	// mainIDs := []string{}

	// for _, result := range mainIDPathResults {
	// 	folder := _getFirstFolder(result.Path)
	// 	mainIDs = append(mainIDs, result.MainID)
	// 	if _, ok := uniqueFolders[folder]; ok {
	// 		continue
	// 	}
	// 	uniqueFolders[folder] = true
	// 	folders = append(folders, folder)
	// }

	// log.Println("[SitemapRows] folders: ", folders)
	// log.Println("[SitemapRows] mainIDs: ", mainIDs)

	// var tmpResults []UserData

	// tmp:= pocketbaseDB.App.Dao().DB().NewQuery().Execute()

	if data.Path == "" || data.Path == "/" {
		err = pocketbaseDB.App.Dao().DB().
			NewQuery(fmt.Sprintf(
				"SELECT data.* FROM %s LEFT JOIN data ON %s.mainID = data.id ORDER BY %s LIMIT %d OFFSET %d",
				db,
				db,
				orderBy(data.Sort, data.ColType),
				data.PerPage,
				data.Page*data.PerPage,
			)).All(&results)
	} else {
		err = pocketbaseDB.App.Dao().DB().
			NewQuery(fmt.Sprintf(
				`
				SELECT data.* FROM %s AS h
				LEFT JOIN data ON h.mainID = data.id
				WHERE
					h.path LIKE '%s' OR
					h.path LIKE '%s/%%' OR
					h.path LIKE '%s?%%' OR
					h.path LIKE '%s#%%'
				ORDER BY %s
				LIMIT %d OFFSET %d`,
				db,
				data.Path,
				data.Path,
				data.Path,
				data.Path,
				orderBy(data.Sort, data.ColType),
				data.PerPage,
				data.Page*data.PerPage,
			)).All(&results)
	}
	// err = pocketbaseDB.App.Dao().DB().
	// 	Select("*").
	// 	From("data").
	// 	Where(dbx.In(
	// 		"id",
	// 		list.ToInterfaceSlice(mainIDs)...,
	// 	)).
	// 	OrderBy(orderBy(data.Sort, data.ColType)).
	// 	Limit(data.PerPage).Offset(data.Page * data.PerPage).
	// 	All(&results)

	log.Println("[SitemapRows] Request: ", data)
	log.Println("[SitemapRows] Response: ", results)

	if err != nil {
		apis.NewBadRequestError("Failed to fetch warehouse items", err)
	}

	return results
}

func (pocketbaseDB *DatabaseAPI) GetData(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: "POST",
		Path:   "api/v1/data",
		Handler: func(c echo.Context) error {

			var data types.GetData
			if err := c.Bind(&data); err != nil {
				return err
			}

			var results []types.UserData2

			if strings.HasPrefix(data.Collection, "http://") || strings.HasPrefix(data.Collection, "https://") {
				results = pocketbaseDB.fetchSitemapRows(data)
			} else {
				results = pocketbaseDB.fetchAllRows(data)
			}

			return c.JSON(http.StatusOK, results)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})

	return nil
}

type Data struct {
	Ids     []string `json:"ids"`
	Page    int64    `json:""`
	PerPage int64    `json:""`
}
