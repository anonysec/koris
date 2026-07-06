package cli

import (
	"bytes"
	"io"
	"net/http"
)

// makeRequest performs an HTTP request to the running panel via the
// safehttp.LocalhostClient (loopback-only, SSRF-protected).
func (c *CLI) makeRequest(method, path string) (*http.Response, error) {
	url := "http://127.0.0.1:8080" + path
	client := c.Client()
	switch method {
	case http.MethodGet:
		return client.Get(url)
	case http.MethodPost:
		return client.Post(url, "application/json", nil)
	default:
		return client.Get(url)
	}
}

// makeRequestWithBody performs an HTTP request with a JSON body.
func (c *CLI) makeRequestWithBody(method, path string, body []byte) (*http.Response, error) {
	url := "http://127.0.0.1:8080" + path
	client := c.Client()
	return client.Post(url, "application/json", bytes.NewReader(body))
}

// drainAndClose reads and discards the response body to allow connection reuse.
func drainAndClose(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
