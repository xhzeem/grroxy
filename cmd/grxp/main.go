package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/glitchedgitz/dadql/dadql"
	"github.com/spf13/cobra"
)

type Param struct {
	Value string `json:"value"`
	Key   string `json:"key"`
	Type  string `json:"type"`
}

type ParsedURL struct {
	Url      string             `json:"url"`
	Scheme   string             `json:"scheme"`
	Host     string             `json:"host"`
	Port     string             `json:"port"`
	TLS      bool               `json:"tls"`
	Path     string             `json:"path"`
	RawPath  string             `json:"raw_path,omitempty"`
	Query    string             `json:"query,omitempty"`
	Params   map[string][]Param `json:"params,omitempty"`
	Fragment string             `json:"fragment,omitempty"`
	User     string             `json:"user,omitempty"`
	Ext      string             `json:"ext,omitempty"`
}

var (
	filter     string
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "grxp",
	Short: "GRXP parses URLs from stdin",
	Long: `GRXP reads URLs from stdin (one per line) and outputs URLs (or JSON with -j flag).

Example:
  cat urls.txt | grxp
  echo "https://example.com/path?key=value" | grxp
  cat urls.txt | grxp -j
  cat urls.txt | grxp -f "scheme = 'https' && tls = true"
  cat urls.txt | grxp -j -f "scheme = 'https'"

Filter Examples (dadql syntax):
  cat urls.txt | grxp -f "scheme = 'https'"
  cat urls.txt | grxp -f "tls = true"
  cat urls.txt | grxp -f "host ~ 'example'"
  cat urls.txt | grxp -f "path ~ '/api'"
  cat urls.txt | grxp -f "port = '443'"
  cat urls.txt | grxp -f "host ~ '%example%'"

Filter Operators:
  =, ==    Equal
  !=       Not equal
  ~        Like/Contains (use %% as wildcard, e.g., '%%example%%')
  !~       Not like
  >, >=    Greater than (or equal)
  <, <=    Less than (or equal)
  AND       AND
  OR       OR
  NOT      NOT

Available fields:
  url, scheme, host, port, tls, path, raw_path, query, fragment, user, ext, params.*`,
	Run: run,
}

func init() {
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "dadql filter expression (e.g., 'scheme = \"https\"')")
	rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON (default: print URL only)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parsed := parseURL(line)

		// Check filter if provided
		if filter != "" {
			if !matchesFilter(parsed) {
				continue
			}
		}

		if jsonOutput {
			jsonData, err := json.Marshal(parsed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
				continue
			}
			fmt.Println(string(jsonData))
		} else {
			fmt.Println(parsed.Url)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

func parseURL(rawURL string) ParsedURL {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ParsedURL{
			Url: rawURL,
		}
	}

	parsed := ParsedURL{
		Url:      rawURL,
		Scheme:   u.Scheme,
		Host:     u.Hostname(),
		Port:     u.Port(),
		Path:     u.Path,
		RawPath:  u.RawPath,
		Query:    u.RawQuery,
		Fragment: u.Fragment,
		TLS:      u.Scheme == "https",
		Ext:      getExtension(u.Path),
	}

	// Only populate Params if there are query parameters
	if u.RawQuery != "" {
		params := make(map[string][]Param)
		for key, values := range u.Query() {
			for _, value := range values {
				paramType := detectType(value)
				params[key] = append(params[key], Param{
					Value: value,
					Key:   key,
					Type:  paramType,
				})
			}
		}
		parsed.Params = params
	}

	if u.User != nil {
		parsed.User = u.User.Username()
	}

	return parsed
}

func detectType(value string) string {
	if value == "" {
		return "string"
	}

	// Try to parse as boolean
	if value == "true" || value == "false" {
		return "boolean"
	}

	// Try to parse as integer (check if it contains a decimal point first)
	if !strings.Contains(value, ".") {
		if _, err := strconv.ParseInt(value, 10, 64); err == nil {
			return "integer"
		}
	}

	// Try to parse as float
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "float"
	}

	return "string"
}

func getExtension(path string) string {
	if path == "" {
		return ""
	}

	// Remove query string and fragment if present
	path = strings.Split(path, "?")[0]
	path = strings.Split(path, "#")[0]

	// Get the last part of the path
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}

	lastPart := parts[len(parts)-1]
	if lastPart == "" {
		return ""
	}

	// Check if it has an extension
	dotIndex := strings.LastIndex(lastPart, ".")
	if dotIndex == -1 || dotIndex == len(lastPart)-1 {
		return ""
	}

	ext := strings.ToLower(lastPart[dotIndex+1:])
	// Remove any query parameters from extension
	ext = strings.Split(ext, "?")[0]
	ext = strings.Split(ext, "#")[0]

	return ext
}

func matchesFilter(parsed ParsedURL) bool {
	if filter == "" {
		return true
	}

	// Convert ParsedURL to map[string]any for dadql
	data := parsedToMap(parsed)

	match, err := dadql.Filter(data, filter)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "[FILTER ERR] %s: %v\n", parsed.Url, err)
		return false
	}

	return match
}

func parsedToMap(parsed ParsedURL) map[string]any {
	data := map[string]any{
		"url":      parsed.Url,
		"scheme":   parsed.Scheme,
		"host":     parsed.Host,
		"port":     parsed.Port,
		"tls":      parsed.TLS,
		"path":     parsed.Path,
		"query":    parsed.Query,
		"fragment": parsed.Fragment,
	}

	if parsed.Ext != "" {
		data["ext"] = parsed.Ext
	}

	if parsed.RawPath != "" {
		data["raw_path"] = parsed.RawPath
	}

	if parsed.User != "" {
		data["user"] = parsed.User
	}

	if len(parsed.Params) > 0 {
		// Convert params to a format that's easier to query
		paramsMap := make(map[string]any)
		for key, params := range parsed.Params {
			paramsMap[key] = params
		}
		data["params"] = paramsMap
	}

	return data
}
