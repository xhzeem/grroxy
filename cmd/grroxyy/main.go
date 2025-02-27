package main

import (
	"errors"
	"net/http"

	"github.com/glitchedgitz/grroxy-db/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "github.com/glitchedgitz/grroxy-db/cmd/grroxyy/migrations"
)

func main() {
	App := pocketbase.NewWithConfig(
		pocketbase.Config{
			ProjectDir:      "D:\\test\\main",
			DefaultDataDir:  "grroxy",
			HideStartBanner: true,
			// DefaultDev: true,
			// DefaultEncryptionEnv: "hJH#GRJ#HG$JH$54h5kjhHJG#JHG#*&Y&EG#F&GIG@JKGH$JHRGJ##JKJH#JHG",
		},
	)

	migratecmd.MustRegister(App, App.RootCmd, migratecmd.Config{})

	App.Bootstrap()

	host, err := utils.CheckAndFindAvailablePort("127.0.0.1:8090")
	if err != nil {
		panic(err)
	}

	_, err = apis.Serve(App, apis.ServeConfig{
		HttpAddr: host,
	})

	if errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
