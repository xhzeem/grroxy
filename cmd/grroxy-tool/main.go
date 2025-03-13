package main

import (
	"errors"
	"flag"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"

	// "github.com/pocketbase/pocketbase/tools/list"
	_ "github.com/glitchedgitz/grroxy-db/cmd/grroxy-tool/migrations"
	"github.com/glitchedgitz/grroxy-db/cmd/grroxy-tool/tools_api"
)

func main() {

	var host string
	var path string
	var name string

	flag.StringVar(&host, "host", "127.0.0.1:8090", "Host address to listen on")
	flag.StringVar(&path, "path", ".", "Project directory path")
	flag.StringVar(&name, "name", "grroxy-tool", "Project name")
	flag.Parse()

	backend := tools_api.Tools{
		App: pocketbase.NewWithConfig(
			pocketbase.Config{
				ProjectDir:      path,
				DefaultDataDir:  name,
				HideStartBanner: true,
			},
		),
		CmdChannel: make(chan tools_api.RunCommandData),
	}

	backend.App.Bootstrap()
	go backend.CommandManager()

	_, err := apis.Serve(backend.App, apis.ServeConfig{
		HttpAddr: host,
	})

	if errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
