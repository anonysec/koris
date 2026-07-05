package cli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// makeRequest performs an HTTP request to the running panel, first trying the
// Unix socket and falling back to HTTP at 127.0.0.1:8080.
func (c *CLI) makeRequest(method, path string) (*http.Response, error) {
	// Try Unix socket first.
	resp, err := c.requestViaSocket(method, path, nil)
	if err == nil {
		return resp, nil
	}

	// Fallback to HTTP.
	return c.requestViaHTTP(method, path, nil)
}

// makeRequestWithBody performs an HTTP request with a JSON body to the running panel,
// first trying the Unix socket and falling back to HTTP at 127.0.0.1:8080.
func (c *CLI) makeRequestWithBody(method, path string, body []byte) (*http.Response, error) {
	// Try Unix socket first.
	resp, err := c.requestViaSocket(method, path, body)
	if err == nil {
		return resp, nil
	}

	// Fallback to HTTP.
	return c.requestViaHTTP(method, path, body)
}

// requestViaSocket attempts the request over the configured Unix socket.
func (c *CLI) requestViaSocket(method, path string, body []byte) (*http.Response, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", c.socketPath, 5*time.Second)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	url := "http://panel" + path
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unix socket request failed: %w", err)
	}
	return resp, nil
}

// requestViaHTTP attempts the request over TCP to 127.0.0.1:8080.
func (c *CLI) requestViaHTTP(method, path string, body []byte) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := "http://127.0.0.1:8080" + path
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	return resp, nil
}
