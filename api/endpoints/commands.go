package endpoints

import (
	"bufio"
	"log"
	"net/http"
	"os/exec"
	"path"

	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/save"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type Cmd struct {
	command    string
	collection string
}

// loop over commandChannel
func (pocketbaseDB *DatabaseAPI) CommandManager() {
	log.Println("[CommandManager Stared]")
	for c := range pocketbaseDB.CmdChannel {
		log.Println("Command received: ", c)
		pocketbaseDB.RunningCommand(c.command, c.collection)
	}
}

func (pocketbaseDB *DatabaseAPI) RunningCommand(command string, collectionName string) {

	log.Println("RunningCommand: ", command)
	cmd := exec.Command("cmd", "/C", command)

	// Create a pipe for the output of the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error creating stdout pipe:", err)
		return
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		log.Println("Error starting command:", err)
		return
	}

	// Create a scanner to read the output line by line
	scanner := bufio.NewScanner(stdout)

	collection, err := pocketbaseDB.App.Dao().FindCollectionByNameOrId(collectionName)
	base.CheckErr("[RunningCommand][FindCollection]:", err)

	// Read the output in real-time
	for scanner.Scan() {
		jsonrow := scanner.Text()
		log.Println("[RunningCommand][Scanner]: ", jsonrow)

		record := models.NewRecord(collection)
		record.Set("data", jsonrow)
		err = pocketbaseDB.App.Dao().SaveRecord(record)
		base.CheckErr("[RunningCommand][SaveRecord]:", err)
	}

	// Wait for the command to finish
	err = cmd.Wait()

	if err != nil {
		log.Println("Error waiting for command:", err)
		return
	}
}

func (pocketbaseDB *DatabaseAPI) RunCommand(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/runcommand",
		Handler: func(c echo.Context) error {

			var data map[string]interface{}
			if err := c.Bind(&data); err != nil {
				return err
			}

			log.Println("[RunCommand]: ", data)

			// send to channel
			pocketbaseDB.CmdChannel <- Cmd{
				command:    data["command"].(string),
				collection: data["collection"].(string),
			}

			return c.String(http.StatusOK, "Created")
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})
	return nil
}

func (pocketbaseDB *DatabaseAPI) SaveFile(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/savefile",
		Handler: func(c echo.Context) error {

			var data map[string]interface{}
			if err := c.Bind(&data); err != nil {
				return err
			}
			log.Println("[SaveFile]: ", data)

			fileName := data["fileName"].(string)
			fileData := data["fileData"].(string)

			filePath := path.Join(pocketbaseDB.Config.CacheDirectory, fileName)

			// Save request_id.txt
			save.WriteFile(filePath, []byte(fileData))

			jsonData := map[string]interface{}{
				"filepath": filePath,
			}

			return c.JSON(http.StatusOK, jsonData)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})
	return nil
}
