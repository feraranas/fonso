package tools

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test 1: ApiCall should successfully make a GET request
func TestApiCall_GET(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a GET request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	// Create JSON input for GET request
	input := ApiCallInput{
		URL:    server.URL,
		Method: "GET",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ApiCall function
	result, err := ApiCall(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result contains status and body
	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200 in result, got: %q", result)
	}
	if !strings.Contains(result, "success") {
		t.Errorf("Expected 'success' in result body, got: %q", result)
	}
}

// Test 2: ApiCall should successfully make a POST request with body
func TestApiCall_POST(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", r.Header.Get("Content-Type"))
		}

		// Read and echo the body
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"received":"` + body["data"] + `"}`))
	}))
	defer server.Close()

	// Create JSON input for POST request
	input := ApiCallInput{
		URL:    server.URL,
		Method: "POST",
		Body:   `{"data":"test payload"}`,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ApiCall function
	result, err := ApiCall(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result contains status and echoed data
	if !strings.Contains(result, "201") {
		t.Errorf("Expected status 201 in result, got: %q", result)
	}
	if !strings.Contains(result, "test payload") {
		t.Errorf("Expected 'test payload' in result, got: %q", result)
	}
}

// Test 3: ApiCall should handle custom headers
func TestApiCall_CustomHeaders(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got: %s", authHeader)
		}

		customHeader := r.Header.Get("X-Custom-Header")
		if customHeader != "custom-value" {
			t.Errorf("Expected X-Custom-Header, got: %s", customHeader)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	// Create JSON input with custom headers
	input := ApiCallInput{
		URL:    server.URL,
		Method: "GET",
		Headers: map[string]string{
			"Authorization":   "Bearer test-token",
			"X-Custom-Header": "custom-value",
		},
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ApiCall function
	result, err := ApiCall(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify successful response
	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200 in result, got: %q", result)
	}
	if !strings.Contains(result, "authenticated") {
		t.Errorf("Expected 'authenticated' in result, got: %q", result)
	}
}

// Test 4: ApiCall should default to GET when method is empty
func TestApiCall_DefaultMethod(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it defaults to GET
		if r.Method != "GET" {
			t.Errorf("Expected GET as default method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// Create JSON input without method (should default to GET)
	input := ApiCallInput{
		URL:    server.URL,
		Method: "", // Empty method
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ApiCall function
	result, err := ApiCall(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify successful response
	if !strings.Contains(result, "200") {
		t.Errorf("Expected status 200 in result, got: %q", result)
	}
}

// Test 5: ApiCall should handle different HTTP methods (PUT, PATCH, DELETE)
func TestApiCall_OtherMethods(t *testing.T) {
	methods := []string{"PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the method
				if r.Method != method {
					t.Errorf("Expected %s request, got %s", method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"method":"` + method + `"}`))
			}))
			defer server.Close()

			// Create JSON input
			input := ApiCallInput{
				URL:    server.URL,
				Method: method,
			}
			inputJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			// Call the ApiCall function
			result, err := ApiCall(inputJSON)

			// Verify no error occurred
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Verify result contains the method
			if !strings.Contains(result, method) {
				t.Errorf("Expected method %s in result, got: %q", method, result)
			}
		})
	}
}
