package main

import (
	"log"
	"os"
	"path"

	// "github.com/pocketbase/dbx"

	"github.com/glitchedgitz/grroxy-db/api/endpoints"
	"github.com/glitchedgitz/grroxy-db/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// "github.com/pocketbase/pocketbase/tools/list"
	_ "github.com/glitchedgitz/grroxy-db/cmd/grroxy-db/migrations"
)

func main() {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Almost never here but panic
		panic(err)
	}

	os.MkdirAll(path.Join(homeDir, ".cache", "grroxy"), 0644)

	// Create an instance of the app structure
	pb := endpoints.DatabaseAPI{
		App: pocketbase.NewWithConfig(
			&pocketbase.Config{
				DefaultDataDir: "grroxy",
			},
		),

		Config: &config.Config{
			CacheDirectory:    path.Join(homeDir, ".cache", "grroxy"),
			ProjectDirectory:  "grroxy_test",
			DatabaseDirectory: "grroxy",
		},

		CmdChannel: make(chan endpoints.Cmd),
	}

	// pb.CmdChannel
	go pb.CommandManager()

	migratecmd.MustRegister(pb.App, pb.App.RootCmd, &migratecmd.Options{
		// enable auto creation of migration files when making collection changes in the Admin UI
		// (the isGoRun check is to enable it only during development)
		// Automigrate: isGoRun,
	})

	// Adding custom endpoints
	pb.App.OnBeforeServe().Add(pb.SitemapNew)
	pb.App.OnBeforeServe().Add(pb.SitemapFetch)
	pb.App.OnBeforeServe().Add(pb.RunCommand)
	pb.App.OnBeforeServe().Add(pb.SendRawRequest)
	pb.App.OnBeforeServe().Add(pb.TextSQL)
	pb.App.OnBeforeServe().Add(pb.SaveFile)
	pb.App.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		pb.App.Dao().DB().NewQuery(`
			DELETE FROM _intercept;
		`).Execute()
		return nil
	})

	if err := pb.App.Start(); err != nil {
		log.Fatal(err)
	}

}
