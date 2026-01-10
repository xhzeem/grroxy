package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/glitchedgitz/dadql/dadql"
	"github.com/glitchedgitz/grroxy-db/grx/rawhttp"
	"github.com/spf13/cobra"
)

var (
	concurrency int
	timeout     float64
	delay       int
	method      string
	verbose     bool
	showStatus  bool
	filter      string
)

var rootCmd = &cobra.Command{
	Use:   "grx",
	Short: "GRX probes URLs and prints those that are alive",
	Long: `GRX reads URLs from stdin and checks which ones respond.
Uses rawhttp with browser TLS fingerprinting to bypass bot detection.

Example:
  cat urls.txt | grx
  echo "https://example.com" | grx
  cat urls.txt | grx -c 100 -t 5

Filter Examples (dadql syntax):
  cat urls.txt | grx -f "resp.status = 200"
  cat urls.txt | grx -f "resp.status >= 200 && resp.status < 300"
  cat urls.txt | grx -f "resp.length > 1000"
  cat urls.txt | grx -f "resp.body ~ 'admin'"
  cat urls.txt | grx -f "resp.body ~ '%login%'"
  cat urls.txt | grx -f "resp.time < 500"
  cat urls.txt | grx -f "resp.mime ~ 'html'"
  cat urls.txt | grx -f "req.host ~ 'api'"

Filter Operators:
  =, ==    Equal
  !=       Not equal
  ~        Like/Contains (use %% as wildcard, e.g., '%%admin%%')
  !~       Not like
  >, >=    Greater than (or equal)
  <, <=    Less than (or equal)
  AND       AND
  OR       OR
  NOT      NOT

Available fields:
  req.method, req.url, req.path, req.host, req.headers.*
  resp.status, resp.length, resp.time, resp.body, resp.mime, resp.headers.*`,
	Run: run,
}

func init() {
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 50, "Number of concurrent requests")
	rootCmd.Flags().Float64VarP(&timeout, "timeout", "t", 10, "Timeout in seconds per request")
	rootCmd.Flags().IntVarP(&delay, "delay", "d", 0, "Delay between requests in milliseconds")
	rootCmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP method to use (GET, HEAD)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show errors")
	rootCmd.Flags().BoolVarP(&showStatus, "status-code", "s", false, "Show status code")
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "dadql filter expression (e.g., 'resp.status = 200')")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	client := rawhttp.NewClient(rawhttp.Config{
		Timeout:            time.Duration(timeout * float64(time.Second)),
		InsecureSkipVerify: true,
	})

	urls := make(chan string, concurrency)
	var wg sync.WaitGroup
	var outputMu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rawURL := range urls {
				data, status, alive := checkURL(client, rawURL)
				if alive && matchesFilter(data, rawURL) {
					outputMu.Lock()
					if showStatus {
						fmt.Printf("[%d] %s\n", status, rawURL)
					} else {
						fmt.Println(rawURL)
					}
					outputMu.Unlock()
				} else if verbose && !alive {
					outputMu.Lock()
					fmt.Fprintf(os.Stderr, "[ERR] %s\n", rawURL)
					outputMu.Unlock()
				}
				if delay > 0 {
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}
			}
		}()
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			urls <- line
		}
	}
	close(urls)

	wg.Wait()
}

func checkURL(client *rawhttp.Client, rawURL string) (map[string]any, int, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, false
	}

	useTLS := u.Scheme == "https"
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if useTLS {
			port = "443"
		} else {
			port = "80"
		}
	}

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	rawReq := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\nAccept: */*\r\n\r\n",
		method, path, host)

	resp, err := client.Send(rawhttp.Request{
		RawBytes: []byte(rawReq),
		Host:     host,
		Port:     port,
		UseTLS:   useTLS,
	})
	if err != nil {
		return nil, 0, false
	}

	parsed := rawhttp.ParseResponse(resp.RawBytes)
	if parsed.Status == 0 {
		return nil, 0, false
	}

	// Build response headers map
	respHeaders := map[string]any{}
	httpHdr := http.Header{}
	for _, header := range parsed.Headers {
		if len(header) >= 2 {
			key := strings.TrimSuffix(header[0], ":")
			val := strings.TrimSpace(header[1])
			httpHdr.Set(key, val)
			respHeaders[key] = val
		}
	}

	// Get content length
	var contentLen int64 = int64(len(parsed.Body))
	if clStr, ok := rawhttp.GetHeaderValue(parsed.Headers, "content-length"); ok {
		if n, err := strconv.ParseInt(strings.TrimSpace(clStr), 10, 64); err == nil {
			contentLen = n
		}
	}

	// Build request headers map
	reqHeaders := map[string]any{
		"Host":       host,
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Accept":     "*/*",
		"Connection": "close",
	}

	data := map[string]any{
		"req": map[string]any{
			"method":  method,
			"url":     path,
			"path":    u.Path,
			"query":   u.RawQuery,
			"host":    host,
			"headers": reqHeaders,
		},
		"resp": map[string]any{
			"status":  parsed.Status,
			"length":  contentLen,
			"mime":    httpHdr.Get("Content-Type"),
			"headers": respHeaders,
			"time":    resp.ResponseTime.Milliseconds(),
			"body":    parsed.Body,
		},
		"host": host,
		"url":  rawURL,
	}

	return data, parsed.Status, true
}

func matchesFilter(data map[string]any, rawURL string) bool {
	if filter == "" {
		return true
	}

	match, err := dadql.Filter(data, filter)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[FILTER ERR] %s: %v\n", rawURL, err)
		}
		return false
	}

	return match
}
