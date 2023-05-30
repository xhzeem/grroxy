package endpoints

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tomnomnom/rawhttp"
)

func sendRawRequest(host, port, rawRequest string) (string, error) {
	// Connect to the server

	// remove http:// or https://
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")

	// Connect to the server
	conn, err := tls.Dial("tcp", host+":"+port, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Send the raw request
	_, err = conn.Write([]byte(rawRequest + "\r\n\r\n"))
	if err != nil {
		return "", fmt.Errorf("failed to write the request: %w", err)
	}

	// Read the response
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return "", fmt.Errorf("failed to read the response: %w", err)
	}

	respString, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return "", err
	}

	return string(respString), nil
}

func sendRawRequest2(rawRequest rawhttp.RawRequest) (string, error) {

	// // Log rawrequest

	resp, err := rawhttp.Do(rawRequest)
	if err != nil {
		return "", fmt.Errorf("an error occurred while making the request: %w", err)
	}

	respString := resp.StatusLine() + "\n"
	for _, h := range resp.Headers() {
		respString += h + "\n"
	}

	respString += "\n" + string(resp.Body()) + "\n"
	return respString, nil
}

func (pocketbaseDB *DatabaseAPI) SendRawRequest(e *core.ServeEvent) error {

	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/sendrawrequest",
		Handler: func(c echo.Context) error {
			var data map[string]interface{}
			if err := c.Bind(&data); err != nil {
				return err
			}

			host := data["host"].(string)
			host = strings.TrimPrefix(host, "http://")
			host = strings.TrimPrefix(host, "https://")

			request := data["request"].(string)
			// replace \n with \r\n

			// request = strings.ReplaceAll(request, "\n", "\r\n") + "\r\n\r\n"

			log.Println("RawRequest TLS: ", data["tls"].(bool))
			log.Println("RawRequest Hostname: ", data["host"].(string))
			log.Println("RawRequest Port: ", data["port"].(string))
			log.Println("RawRequest Timeout: ", time.Duration(data["timeout"].(float64))*time.Second)
			log.Println("RawRequest Request: ", request)

			// respString, err := sendRawRequest2(rawhttp.RawRequest{
			// 	TLS:      data["tls"].(bool),
			// 	Hostname: host,
			// 	Port:     data["port"].(string),
			// 	Request:  request,
			// 	Timeout:  time.Duration(data["timeout"].(float64)) * time.Second,
			// })

			respString, err := sendRawRequest(data["host"].(string), data["port"].(string), data["request"].(string))
			if err != nil {
				return err
			}

			response := map[string]interface{}{
				"response": respString,
			}

			return c.JSON(http.StatusOK, response)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(pocketbaseDB.App),
		},
	})
	return nil
}
