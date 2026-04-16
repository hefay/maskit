package maskit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONDeserializer_Deserialize(t *testing.T) {
	jsonPayload := `{"JobId": "job-12345"}`
	reader := strings.NewReader(jsonPayload)

	deserializer := &JSONDeserializer{}
	resp, err := deserializer.Deserialize(reader)

	if err != nil {
		t.Fatalf("Expected success, but got an error: %v", err)
	}

	if resp.JobID != "job-12345" {
		t.Errorf("Expected JobID to be 'job-12345', got '%s'", resp.JobID)
	}
}

func TestMultipartSerializer_Serialize(t *testing.T) {
	// Prepare test data
	mockImageContent := "fake-image-data"
	req := MaskingRequest{
		Image:         strings.NewReader(mockImageContent),
		Faces:         true,
		BlurStrength:  50,
		EdgeBlurSize:  1.5,
		UseWebhook:    true,
		Metadata:      "test-meta",
	}

	serializer := &MultipartSerializer{}
	bodyReader, contentType, err := serializer.Serialize(req)

	if err != nil {
		t.Fatalf("Expected serialization to succeed, got error: %v", err)
	}

	if !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Errorf("Expected Content-Type starting with 'multipart/form-data', got '%s'", contentType)
	}

	// Read the entire request body to verify its contents
	bodyBytes, _ := io.ReadAll(bodyReader)
	bodyString := string(bodyBytes)

	// Check if it contains the correct camelCase keys and our values
	expectedStrings := []string{
		`name="image"; filename="upload.jpg"`,
		mockImageContent,
		`name="faces"`,
		"blur",
		`name="blurStrength"`,
		"50",
		`name="edgeBlurSize"`,
		"1.500000",
		`name="useWebhook"`,
		"true",
		`name="metadata"`,
		"test-meta",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(bodyString, expected) {
			t.Errorf("Expected body to contain '%s', but it wasn't found. Body: %s", expected, bodyString)
		}
	}
}

func TestHTTPTransport_Send_Success(t *testing.T) {
	// 1. Create a test (mock) server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the server received the correct method and Content-Type
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Invalid Content-Type: %s", r.Header.Get("Content-Type"))
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"JobId": "mock-job-999"}`))
	}))
	defer mockServer.Close() // Don't forget to close the server after the test

	// 2. Configure the transport
	transport := &HTTPTransport{
		Client: mockServer.Client(), // Use the client from the test server
	}

	// 3. Execute the method under test
	req := MaskingRequest{
		Image: strings.NewReader("dummy-image"),
	}
	resp, err := transport.Send(mockServer.URL, req)

	// 4. Verify the results
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if resp.JobID != "mock-job-999" {
		t.Errorf("Expected JobID to be 'mock-job-999', got '%s'", resp.JobID)
	}
}

func TestHTTPTransport_Send_APIError(t *testing.T) {
	// Set up the mock server to return an HTTP error code (e.g., 400 Bad Request)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid image format"}`))
	}))
	defer mockServer.Close()

	transport := &HTTPTransport{
		Client: mockServer.Client(),
	}

	req := MaskingRequest{
		Image: strings.NewReader("bad-data"),
	}
	_, err := transport.Send(mockServer.URL, req)

	if err == nil {
		t.Fatal("Expected an error due to 400 Bad Request, but none occurred")
	}

	// Verify that the error message contains the status code and the server message
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid image format") {
		t.Errorf("Error does not contain the expected HTTP status information. Got: %v", err)
	}
}
