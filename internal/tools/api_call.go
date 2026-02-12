// api_call.go - HTTP API caller tool implementation
package tools

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// ApiCallInput defines parameters for api_call tool
type ApiCallInput struct {
	URL     string            `json:"url" jsonschema_description:"The full URL to call (e.g. https://127.0.0.1:8080/insert)"`
	Method  string            `json:"method" jsonschema_description:"HTTP method: GET, POST, PUT, PATCH, or DELETE"`
	Body    string            `json:"body" jsonschema_description:"Optional request body (JSON string). Omit for GET requests."`
	Headers map[string]string `json:"headers" jsonschema_description:"Optional HTTP headers as key-value pairs"`
}

// ApiCallInputSchema is the generated JSON schema for ApiCallInput
var ApiCallInputSchema = GenerateSchema[ApiCallInput]()

// ApiCallDefinition allows Claude to make HTTP API calls
var ApiCallDefinition = ToolDefinition{
	Name: "api_call",
	Description: `Make an HTTP request to any API. Use for REST APIs, webhooks, or backend services.
Examples: POST JSON to insert data, GET to fetch, etc.
For localhost/127.0.0.1 with HTTPS, self-signed certs are accepted.`,
	InputSchema: ApiCallInputSchema,
	Function:    ApiCall,
}

// How it works:
// 1. Parse the API call request (URL, method, body, headers)
// 2. Make the HTTP request using net/http
// 3. Return the response body to Claude
// 4. Handle errors and return clear error messages
// 5. Log all API calls for security auditing

func ApiCall(input json.RawMessage) (string, error) {
	var req ApiCallInput
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	// Default to GET if empty
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}

	// Set the body
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewReader([]byte(req.Body))
	}

	// Create the HTTP request
	httpReq, err := http.NewRequest(method, req.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	// Set the content type to JSON if not set
	if req.Body != "" && req.Headers["Content-Type"] == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Allow HTTPS with self-signed certs (common for local dev)
	// InsecureSkipVerify: true is used so local HTTPS (e.g. 127.0.0.1) works with self-signed certs. Use only for local/dev.
	// TODO: Security	Restrict URLs, add auth headers, or only allow certain hosts if you expose this beyond local dev.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Make the HTTP request
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	result := fmt.Sprintf("Status: %d %s\nBody: %s", resp.StatusCode, resp.Status, string(respBody))
	log.Printf("api_call %s %s -> %d", method, req.URL, resp.StatusCode)
	return result, nil
}
