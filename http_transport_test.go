package maskit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONDeserializer_Deserialize(t *testing.T) {
	reader := strings.NewReader(`{"JobId": "job-12345"}`)

	resp, err := (&JSONDeserializer{}).Deserialize(reader)

	require.NoError(t, err)
	assert.Equal(t, "job-12345", resp.JobID)
}

func TestMultipartSerializer_Serialize(t *testing.T) {
	req := MaskingRequest{
		Image:        strings.NewReader("fake-image-data"),
		Faces:        true,
		BlurStrength: 50,
		EdgeBlurSize: 1.5,
		UseWebhook:   true,
		Metadata:     "test-meta",
	}

	bodyReader, contentType, err := (&MultipartSerializer{}).Serialize(req)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(contentType, "multipart/form-data"))

	bodyBytes, _ := io.ReadAll(bodyReader)
	bodyString := string(bodyBytes)

	expected := []string{
		`name="image"; filename="upload.jpg"`,
		"fake-image-data",
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
	for _, s := range expected {
		assert.Contains(t, bodyString, s)
	}
}

func TestHTTPTransport_Send_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"JobId": "mock-job-999"}`))
	}))
	defer mockServer.Close()

	resp, err := (&HTTPTransport{Client: mockServer.Client()}).Send(mockServer.URL, MaskingRequest{
		Image: strings.NewReader("dummy-image"),
	})

	require.NoError(t, err)
	assert.Equal(t, "mock-job-999", resp.JobID)
}

func TestHTTPTransport_Send_APIError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid image format"}`))
	}))
	defer mockServer.Close()

	_, err := (&HTTPTransport{Client: mockServer.Client()}).Send(mockServer.URL, MaskingRequest{
		Image: strings.NewReader("bad-data"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "invalid image format")
}

func TestHTTPTransport_Send_SetsAPIKeyHeader(t *testing.T) {
	var actualKey string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualKey = r.Header.Get(APIKeyHeader)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"JobId": "job-1"}`))
	}))
	defer mockServer.Close()

	_, _ = (&HTTPTransport{
		Client: mockServer.Client(),
		APIKey: "my-secret-key",
	}).Send(mockServer.URL, MaskingRequest{Image: strings.NewReader("img")})

	assert.Equal(t, "my-secret-key", actualKey)
}

func TestHTTPTransport_GetJobStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "job-123", r.URL.Query().Get("jobid"))
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"JobId": "job-123", "Status": "readytodownload"}`))
		}))
		defer mockServer.Close()

		resp, err := (&HTTPTransport{Client: mockServer.Client()}).GetJobStatus(mockServer.URL + "?jobid=job-123")

		require.NoError(t, err)
		assert.Equal(t, "job-123", resp.JobID)
		assert.Equal(t, JobStatusReadyToDownload, resp.Status)
	})

	t.Run("not found", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "job not found"}`))
		}))
		defer mockServer.Close()

		_, err := (&HTTPTransport{Client: mockServer.Client()}).GetJobStatus(mockServer.URL)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Contains(t, err.Error(), "job not found")
	})
}

func TestHTTPTransport_DownloadImage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedData := []byte("fake-jpeg-binary-data")
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "job-456", r.URL.Query().Get("jobid"))
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expectedData)
		}))
		defer mockServer.Close()

		reader, err := (&HTTPTransport{Client: mockServer.Client()}).DownloadImage(mockServer.URL + "?jobid=job-456")
		require.NoError(t, err)
		defer reader.Close()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, expectedData, data)
	})

	t.Run("not ready", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "image is not ready for download"}`))
		}))
		defer mockServer.Close()

		_, err := (&HTTPTransport{Client: mockServer.Client()}).DownloadImage(mockServer.URL)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "not ready")
	})
}
