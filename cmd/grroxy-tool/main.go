package main

import (
	"errors"
	"flag"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"

	// "github.com/pocketbase/pocketbase/tools/list"
	_ "github.com/glitchedgitz/grroxy-db/cmd/grroxy/migrations"
)

func main() {

	var host string
	var path string
	var name string

	flag.StringVar(&host, "host", "127.0.0.1:8090", "Host address to listen on")
	flag.StringVar(&path, "path", ".", "Project directory path")
	flag.StringVar(&name, "name", "grroxy", "Project name")
	flag.Parse()

	App := pocketbase.NewWithConfig(
		pocketbase.Config{
			ProjectDir:      path,
			DefaultDataDir:  name,
			HideStartBanner: true,
			// DefaultDev: true,
			// DefaultEncryptionEnv: "hJH#GRJ#HG$JH$54h5kjhHJG#JHG#*&Y&EG#F&GIG@JKGH$JHRGJ##JKJH#JHG",
		},
	)

	App.Bootstrap()

	_, err := apis.Serve(App, apis.ServeConfig{
		HttpAddr: host,
	})

	if errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
