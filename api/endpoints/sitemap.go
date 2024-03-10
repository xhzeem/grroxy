package endpoints

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/schemas"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/jpillora/go-tld"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	wappalyzer "github.com/projectdiscovery/wappalyzergo"
)

func (pocketbaseDB *DatabaseAPI) SitemapNew(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/sitemap/new",
		Handler: func(c echo.Context) error {
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			recordd, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			isGuest := admin == nil && recordd == nil

			if isGuest {
				return c.String(http.StatusForbidden, "")
			}

			var data types.SitemapGet
			var wg sync.WaitGroup

			if err := c.Bind(&data); err != nil {
				return err
			}
			log.Print("SitemapNew: ", data)

			var collectionExists = true

			SitemapCollectionName := base.ParseDatabaseName(data.Host)
			err := pocketbaseDB.CreateCollection(SitemapCollectionName, schemas.Sitemap)

			// Checking error if it is collection already exists
			// This is the error "constraint failed: UNIQUE constraint failed: collections.name (2067)"
			if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				collectionExists = true
			} else {
				collectionExists = false
			}

			// New Host
			go func() {
				if !collectionExists {
					wg.Add(1)
					defer wg.Done()
					// Fetch fingerprints
					resp, err := http.DefaultClient.Get(data.Host)
					var fingerprints map[string]struct{} = make(map[string]struct{})
					var respData []byte = []byte("0")
					var jsonBytes []byte = []byte("0")
					var status int = 0
					// Fingerprint to json
					jsonString := "{}"

					if err != nil {
						log.Println("[http.DefaultClient.Get]: ", err)
					} else {
						respData, err = io.ReadAll(resp.Body) // Ignoring error for example
						if err != nil {
							log.Println(err)
						} else {
							status = resp.StatusCode
							wappalyzerClient, err := wappalyzer.New()
							if err != nil {
								log.Println("Wappylyzer Error: ", err)
							} else {

								// Todo: Create a custom wappylyzer to give back the logo and accent color of tech

								fingerprints = wappalyzerClient.Fingerprint(resp.Header, respData)
								jsonBytes, err = json.Marshal(fingerprints)
								if err != nil {
									log.Println(err)
								} else {
									jsonString = string(jsonBytes)
								}
								fmt.Printf("Wappylyzer Fingerprints %v\n", fingerprints)
							}
						}
					}

					// Insert row in _hosts
					u, _ := tld.Parse(data.Host)
					// title, _ := "", ""
					title, _ := base.ExtractTitle(respData)

					// Instead of searching every time, we might store it in DatabaseAPI
					collection, err := pocketbaseDB.App.Dao().FindCollectionByNameOrId("_hosts")
					if err != nil {
						log.Println("Error: ", err)
					}

					record := models.NewRecord(collection)

					record.Set("host", data.Host)
					record.Set("smartsort", base.SmartSort(data.Host))
					record.Set("domain", u.Domain+"."+u.TLD)
					record.Set("status", status)
					record.Set("title", title)
					record.Set("tech", jsonString)

					err = pocketbaseDB.App.Dao().Save(record)

					if err != nil {
						log.Println("Error: ", err)
					}
				}
			}()

			// Inserting endpoint data
			collection, err := pocketbaseDB.App.Dao().FindCollectionByNameOrId(SitemapCollectionName)
			if err != nil {
				log.Println("Error: ", err)
			}

			record := models.NewRecord(collection)

			record.Set("id", data.Data)
			record.Set("path", data.Path)
			record.Set("query", data.Query)
			record.Set("fragment", data.Fragment)
			record.Set("type", data.Type)
			record.Set("ext", data.Ext)
			record.Set("data", data.Data)
			err = pocketbaseDB.App.Dao().Save(record)

			if err != nil {
				log.Println("Error: ", err)
			}

			// log.Println("Executed: ", result)

			if err != nil {
				// return nil
				log.Println("Error: ", err)
				// apis.NewBadRequestError("Failed to create collection", err)
			}

			wg.Wait()

			return c.String(http.StatusOK, "Created")
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})
	return nil
}

func (pocketbaseDB *DatabaseAPI) SitemapFetch(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/sitemap/fetch",
		Handler: func(c echo.Context) error {
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			recordd, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			isGuest := admin == nil && recordd == nil

			if isGuest {
				return c.String(http.StatusForbidden, "")
			}

			var data types.SitemapFetch
			if err := c.Bind(&data); err != nil {
				return err
			}

			db := base.ParseDatabaseName(data.Host)

			// Regex: '^path/([^/]+\s*)?$'
			// regexQuery := fmt.Sprintf(`^%s/([^/]+\s*)?$`, data.Path)

			// Simplier for noeWHERE path LIKE '/s/%'
			regexQuery := data.Path + `/%`

			var result []types.SitemapGet
			// var tmpResult []map[string]interface{}
			uniqueMap := make(map[string]map[string]interface{})
			var titles []string
			var err error

			if data.Path == "" {
				err = pocketbaseDB.App.Dao().DB().NewQuery("SELECT * FROM " + db).All(&result)
			} else {
				err = pocketbaseDB.App.Dao().DB().NewQuery("SELECT * FROM " + db + " WHERE path LIKE '" + regexQuery + "'").All(&result)
			}

			for _, item := range result {
				tmpPath := strings.TrimPrefix(item.Path, data.Path)
				tmpPath = strings.TrimPrefix(tmpPath, "/")

				var part string
				if index := strings.IndexAny(tmpPath, "?#"); index != -1 {
					part = tmpPath[:index]
				} else {
					part = tmpPath
				}

				title := strings.Split(part, "/")[0]

				if _, exists := uniqueMap[title]; !exists {
					uniqueMap[title] = map[string]interface{}{
						"host":  data.Host,
						"path":  data.Path + "/" + title,
						"type":  item.Type,
						"title": title,
						"ext":   item.Ext,
						"query": item.Query,
					}
					titles = append(titles, title)
				}
			}

			sort.Strings(titles)
			var tmpResult2 []map[string]interface{}
			for _, title := range titles {
				tmpResult2 = append(tmpResult2, uniqueMap[title])
			}
			log.Println("[SitemapFetch] Request: ", data)
			log.Println("[SitemapFetch] Response: ", tmpResult2)

			if err != nil {
				apis.NewBadRequestError("Failed to fetch warehouse items", err)
			}

			return c.JSON(http.StatusOK, tmpResult2)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})

	return nil
}
