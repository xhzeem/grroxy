package endpoints

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tomnomnom/rawhttp"
	"golang.org/x/net/http2"
)

func sendRawRequest(data rawhttp.RawRequest) (string, error) {
	// Connect to the server
	var host = data.Hostname
	var port = data.Port
	var rawRequest = data.Request
	var addr = host + ":" + port

	log.Println("Addr: ", addr)

	var conn net.Conn
	var err error
	if data.TLS {
		conn, err = tls.DialWithDialer(&net.Dialer{
			Timeout: data.Timeout,
		}, "tcp", addr, &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = net.DialTimeout("tcp", addr, data.Timeout)
	}
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Send the raw request
	_, err = fmt.Fprintf(conn, "%s\r\n\r\n", rawRequest)
	if err != nil {
		return "", fmt.Errorf("failed to write the request: %w", err)
	}

	// Read the response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return "", fmt.Errorf("failed to read the response: %w", err)
	}
	defer resp.Body.Close()

	respString, err := base.ResponseToString(resp)
	base.CheckErr("[sendRawRequest] ", err)

	return respString, nil
}

// func sendRawRequest(data rawhttp.RawRequest) (string, error) {
// 	// Connect to the server

// 	var host = data.Hostname
// 	var port = data.Port
// 	var rawRequest = data.Request

// 	// remove http:// or https://
// 	// host = strings.TrimPrefix(host, "http://")
// 	// host = strings.TrimPrefix(host, "https://")

// 	var addr = host + ":" + port
// 	// if data.TLS {
// 	// 	addr = "https://" + addr
// 	// } else {
// 	// 	addr = "http://" + addr
// 	// }

// 	log.Println("Addr: ", addr)

// 	var conn *tls.Conn
// 	var err error
// 	if data.TLS {
// 		conn, err = tls.DialWithDialer(&net.Dialer{
// 			Timeout: data.Timeout,
// 		}, "tcp", addr, &tls.Config{
// 			InsecureSkipVerify: true,
// 		})
// 	} else {
// 		conn, err = tls.DialWithDialer(&net.Dialer{
// 			Timeout: data.Timeout,
// 		}, "tcp", addr, &tls.Config{})
// 	}

// 	// Connect to the server
// 	// conn, err := tls.DialWithDialer(&net.Dialer{
// 	// 	Timeout: data.Timeout,
// 	// }, "tcp", addr, &tls.Config{
// 	// 	InsecureSkipVerify: true,
// 	// })

// 	if err != nil {
// 		return "", err
// 	}
// 	defer conn.Close()

// 	// Send the raw request
// 	_, err = conn.Write([]byte(rawRequest + "\r\n\r\n"))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to write the request: %w", err)
// 	}

// 	// Read the response
// 	reader := bufio.NewReader(conn)

// 	resp, err := http.ReadResponse(reader, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read the response: %w", err)
// 	}

// 	respString, err := httputil.DumpResponse(resp, true)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(respString), nil
// }

// for http2
func sendRawRequest1(data rawhttp.RawRequest) (string, error) {
	var host = data.Hostname
	var port = data.Port
	var rawRequest = data.Request

	var addr = host + ":" + port

	log.Println("Addr: ", addr)

	var conn net.Conn
	var err error
	if data.TLS {
		conn, err = tls.Dial("tcp", addr, &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create an HTTP/2 client
	tlsConn := conn.(*tls.Conn)
	err = tlsConn.Handshake()
	if err != nil {
		return "", fmt.Errorf("TLS handshake failed: %w", err)
	}
	transport := &http2.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: transport,
	}

	// Create an HTTP request with the raw payload
	req, err := http.NewRequest("GET", "https://"+host+":"+port, strings.NewReader(rawRequest))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request and trace the events
	trace := &httptrace.ClientTrace{}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respString, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return "", fmt.Errorf("failed to read the response: %w", err)
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

			mappedData := rawhttp.RawRequest{
				TLS:      data["tls"].(bool),
				Hostname: host,
				Port:     data["port"].(string),
				Request:  request,
				Timeout:  time.Duration(data["timeout"].(float64)) * time.Second,
			}

			// respString, err := sendRawRequest2(mappedData)
			respString, err := sendRawRequest(mappedData)

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
