package maskit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// RequestSerializer defines the interface for serializing a MaskingRequest
// into a format suitable for an HTTP request body.
type RequestSerializer interface {
	// Serialize takes a MaskingRequest and returns the request body as an io.Reader,
	// the Content-Type header string, and an error if serialization fails.
	Serialize(payload MaskingRequest) (io.Reader, string, error)
}

// ResponseDeserializer defines the interface for decoding the HTTP response body
// into a MaskingResponse struct.
type ResponseDeserializer interface {
	// Deserialize reads from an io.Reader and decodes it into a MaskingResponse.
	Deserialize(r io.Reader) (MaskingResponse, error)
}

// MultipartSerializer implements RequestSerializer using multipart/form-data.
type MultipartSerializer struct{}

// Serialize builds the multipart/form-data payload with camelCase field names.
func (s *MultipartSerializer) Serialize(payload MaskingRequest) (io.Reader, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 1. Append the image file if provided
	if payload.Image != nil {
		part, err := writer.CreateFormFile("image", "upload.jpg")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file: %w", err)
		}
		if _, err := io.Copy(part, payload.Image); err != nil {
			return nil, "", fmt.Errorf("failed to copy image content: %w", err)
		}
	}

	// 2. Append additional metadata and configuration fields using camelCase
	_ = writer.WriteField("faces", fmt.Sprintf("%v", payload.Faces))
	_ = writer.WriteField("humans", fmt.Sprintf("%v", payload.Humans))
	_ = writer.WriteField("licensePlates", fmt.Sprintf("%v", payload.LicensePlates))
	_ = writer.WriteField("shape", fmt.Sprintf("%v", payload.Shape))
	_ = writer.WriteField("method", fmt.Sprintf("%v", payload.Method))
	_ = writer.WriteField("blurStrength", fmt.Sprintf("%d", payload.BlurStrength))
	_ = writer.WriteField("edgeBlurSize", fmt.Sprintf("%f", payload.EdgeBlurSize))
	_ = writer.WriteField("useWebhook", fmt.Sprintf("%t", payload.UseWebhook))

	if payload.Metadata != "" {
		_ = writer.WriteField("metadata", payload.Metadata)
	}

	// Close the writer to finalize the multipart boundary
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

// JSONDeserializer implements ResponseDeserializer using standard JSON decoding.
type JSONDeserializer struct{}

// Deserialize decodes a JSON payload into a MaskingResponse.
func (d *JSONDeserializer) Deserialize(r io.Reader) (MaskingResponse, error) {
	var maskingResp MaskingResponse
	if err := json.NewDecoder(r).Decode(&maskingResp); err != nil {
		return MaskingResponse{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return maskingResp, nil
}

// HTTPTransport handles the low-level communication with the masking API.
type HTTPTransport struct {
	// Client is the underlying HTTP client used to execute requests.
	Client *http.Client

	// Serializer converts the MaskingRequest into an HTTP request body.
	Serializer RequestSerializer

	// Deserializer parses the HTTP response body into a MaskingResponse.
	Deserializer ResponseDeserializer

	// APIKey is the key sent in the X-Api-Key header for authentication.
	APIKey string
}

func (t *HTTPTransport) client() *http.Client {
	if t.Client != nil {
		return t.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (t *HTTPTransport) setAuth(req *http.Request) {
	if t.APIKey != "" {
		req.Header.Set(APIKeyHeader, t.APIKey)
	}
}

func (t *HTTPTransport) doRequest(req *http.Request) (*http.Response, error) {
	t.setAuth(req)
	resp, err := t.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

// Send executes the masking request utilizing the configured Serializer and Deserializer.
func (t *HTTPTransport) Send(url string, payload MaskingRequest) (MaskingResponse, error) {
	// Use default serializer if none is provided
	serializer := t.Serializer
	if serializer == nil {
		serializer = &MultipartSerializer{}
	}

	// Use default deserializer if none is provided
	deserializer := t.Deserializer
	if deserializer == nil {
		deserializer = &JSONDeserializer{}
	}

	// 1. Serialize the request payload
	bodyReader, contentType, err := serializer.Serialize(payload)
	if err != nil {
		return MaskingResponse{}, fmt.Errorf("serialization failed: %w", err)
	}

	// 2. Create the HTTP POST request
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return MaskingResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := t.doRequest(req)
	if err != nil {
		return MaskingResponse{}, err
	}
	defer resp.Body.Close()

	maskingResp, err := deserializer.Deserialize(resp.Body)
	if err != nil {
		return MaskingResponse{}, fmt.Errorf("deserialization failed: %w", err)
	}

	return maskingResp, nil
}

// GetJobStatus calls the image-status endpoint and returns the current job status.
func (t *HTTPTransport) GetJobStatus(url string) (ImageStatusResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ImageStatusResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := t.doRequest(req)
	if err != nil {
		return ImageStatusResponse{}, err
	}
	defer resp.Body.Close()

	var status ImageStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return ImageStatusResponse{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return status, nil
}

// DownloadImage calls the image-download endpoint and returns the masked image as a stream.
func (t *HTTPTransport) DownloadImage(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
