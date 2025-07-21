package grrhttp

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/glitchedgitz/grroxy-db/utils"
)

func DecompressResponse(reader io.Reader, contentEncoding string) (io.Reader, error) {
	switch strings.ToLower(contentEncoding) {
	case "gzip", "x-gzip":
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			// If decompression fails, return the original reader
			return reader, nil
		}
		return gzReader, nil
	case "br", "brotli":
		return brotli.NewReader(reader), nil
	case "deflate":
		// Note: deflate is not implemented here, would need zlib
		// For now, return original reader
		return reader, nil
	default:
		return reader, nil
	}
}

func DumpResponse(resp *http.Response) string {

	var err error
	var bodyReader io.Reader

	// Always check for compression and decompress if needed
	contentEncoding := resp.Header.Get("Content-Encoding")
	bodyReader, err = DecompressResponse(resp.Body, contentEncoding)
	utils.CheckErr("[DumpResponse] Decompression error: ", err)

	// Read the decompressed body
	bf := bufio.NewReader(bodyReader)
	respbody, err := io.ReadAll(bf)
	utils.CheckErr("[DumpResponse] Read body error: ", err)

	// Update content length to reflect decompressed size
	cl := int64(len(respbody))

	finalResp := fmt.Sprintf("%s %s\n", resp.Proto, resp.Status)

	// Build headers, but remove Content-Encoding since we've decompressed
	hasContentLength := false
	for header, value := range resp.Header {
		// Skip Content-Encoding header since we've already decompressed
		if strings.EqualFold(header, "Content-Encoding") {
			continue
		}

		// Update Content-Length to reflect decompressed size
		if strings.EqualFold(header, "Content-Length") {
			finalResp += fmt.Sprintf("%s: %d\n", header, cl)
			hasContentLength = true
		} else {
			for _, val := range value {
				finalResp += fmt.Sprintf("%s: %s\n", header, val)
			}
		}
	}

	// Add Content-Length if it didn't exist in the original response
	if !hasContentLength {
		finalResp += fmt.Sprintf("Content-Length: %d\n", cl)
	}

	finalResp += "\n" + string(respbody)

	return finalResp
}

// 	bf := bufio.NewReader(bodyReader)

// 	finalResp := fmt.Sprintf("%s %s\n", resp.Proto, resp.Status)

// 	for header, value := range resp.Header {
// 		finalResp += fmt.Sprintf("%s: %s\n", header, value)
// 	}

// 	if respbody, err := io.ReadAll(bf); err == nil {
// 		resp.ContentLength = int64(len(respbody))
// 		resp.Body = io.NopCloser(bytes.NewReader(respbody))
// 		finalResp, err := httputils.DumpResponse(resp, true)
// 		utils.CheckErr("[DumpResponse]", err)
// 		return string(finalResp)
// 	} else {
// 		log.Println("Error reading response body:", err)
// 		return ""
// 	}
// }
