package api

import (
	"log"
	"net/http"
	"os"

	"github.com/glitchedgitz/grroxy-db/browser"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// DownloadCert serves the unified CA certificate (ca.crt) for download
// All certificates now managed by rawproxy system
func (backend *Backend) DownloadCert(e *core.ServeEvent) error {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/cacert.crt",
		Handler: func(c echo.Context) error {
			// Get certificate path from unified system
			var certPath string

			// Priority 1: If proxy is running, use its certificate
			if PROXY != nil {
				certPath = PROXY.GetCertPath()
				log.Printf("[Certificate] Serving certificate from running proxy: %s", certPath)
			} else {
				// Priority 2: Generate/get certificate using unified system
				filePath, err := browser.GenerateCert(backend.Config.ConfigDirectory)
				if err != nil {
					log.Printf("[Certificate] Error getting certificate: %v", err)
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Failed to get certificate",
					})
				}
				certPath = filePath
				log.Printf("[Certificate] Serving certificate from storage: %s", certPath)
			}

			// Verify certificate exists
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				log.Printf("[Certificate] Certificate file not found: %s", certPath)
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "Certificate file not found. Please start the proxy first.",
				})
			}

			// Serve the certificate file
			return c.Attachment(certPath, "grroxy-ca.crt")
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(backend.App),
		},
	})
	return nil
}
