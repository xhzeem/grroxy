package api

import (
	"fmt"
	"net/http"

	"github.com/glitchedgitz/grroxy-db/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func (backend *Backend) Tools(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: "GET",
		Path:   "/api/tool",
		Handler: func(c echo.Context) error {
			path := c.QueryParam("path")
			// backend.App.Bootstrap()
			hostAddress, err := utils.CheckAndFindAvailablePort("127.0.0.1:8090")

			fmt.Println("path", path)
			fmt.Println("hostAddress", hostAddress)
			fmt.Println("err", err)

			var NewApp = pocketbase.NewWithConfig(
				pocketbase.Config{
					ProjectDir:      path,
					DefaultDataDir:  "weird",
					HideStartBanner: true,
					// DefaultDev: true,
					// DefaultEncryptionEnv: "hJH#GRJ#HG$JH$54h5kjhHJG#JHG#*&Y&EG#F&GIG@JKGH$JHRGJ##JKJH#JHG",
				},
			)

			NewApp.Bootstrap()


			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}

			_, err = apis.Serve(NewApp, apis.ServeConfig{
				HttpAddr: hostAddress,
			})

			fmt.Println("err", err)

			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}

			return c.String(http.StatusOK, fmt.Sprintf("Path parameter: %s", path))
		},
	})
	return nil
}
