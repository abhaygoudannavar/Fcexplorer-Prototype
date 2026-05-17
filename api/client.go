package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client talks to the Firecracker API over a Unix domain socket.
// Firecracker doesn't support HTTP/2, so we stick with HTTP/1.1.
type Client struct {
	socketPath string
	httpClient *http.Client
	dryRun     bool
}

func NewClient(socketPath string, dryRun bool) *Client {
	// Custom transport that dials the Unix socket instead of TCP.
	// This is the "proper" way to do HTTP over Unix sockets in Go.
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, 3*time.Second)
		},
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{Transport: transport},
		dryRun:     dryRun,
	}
}

// Request sends an HTTP request to the Firecracker API.
// method is GET, PUT, etc. path is like "/machine-config".
// body can be nil for GET requests.
func (c *Client) Request(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(data)

		if c.dryRun {
			fmt.Printf("[DRY RUN] %s %s\n", method, path)
			fmt.Printf("  Body: %s\n\n", string(data))
			return nil, 200, nil
		}
	} else if c.dryRun {
		fmt.Printf("[DRY RUN] %s %s\n\n", method, path)
		return nil, 200, nil
	}

	// Firecracker expects requests to http://localhost, the actual
	// routing happens through the socket file, not the hostname.
	url := "http://localhost" + path

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %v", err)
	}

	// Firecracker expects Content-Type to be application/json for all PUT requests
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %v", err)
	}

	return respBody, resp.StatusCode, nil
}

// RawRequest demonstrates the "manual" approach — building a raw HTTP/1.1
// request and sending it over the socket with net.Dial.
// I wrote this to understand what http.Client is doing under the hood.
// In practice you'd use the Request() method above, but this shows the
// actual bytes going over the wire.
func (c *Client) RawRequest(method, path string, body []byte) (string, error) {
	if c.dryRun {
		raw := buildRawHTTP(method, path, body)
		fmt.Printf("[DRY RUN] Raw HTTP request:\n%s\n", raw)
		return "", nil
	}

	conn, err := net.DialTimeout("unix", c.socketPath, 3*time.Second)
	if err != nil {
		return "", fmt.Errorf("dial socket: %v", err)
	}
	defer conn.Close()

	raw := buildRawHTTP(method, path, body)

	// fmt.Printf("DEBUG: sending %d bytes\n", len(raw))
	_, err = conn.Write([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("write to socket: %v", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read from socket: %v", err)
	}

	return string(buf[:n]), nil
}

// buildRawHTTP constructs a raw HTTP/1.1 request string.
// This is what actually goes over the Unix socket.
func buildRawHTTP(method, path string, body []byte) string {
	req := fmt.Sprintf("%s %s HTTP/1.1\r\n", method, path)
	req += "Host: localhost\r\n"
	req += "Accept: application/json\r\n"

	if body != nil {
		req += "Content-Type: application/json\r\n"
		req += fmt.Sprintf("Content-Length: %d\r\n", len(body))
	}

	req += "\r\n"

	if body != nil {
		req += string(body)
	}

	return req
}
