package maskit

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOneImage(t *testing.T) {
	transport := NewMockTransport(t)
	testClient := createTestClient(transport)

	var capturedRequest MaskingRequest

	transport.EXPECT().
		Send("https://app.maskit.ai/api/v1/masking/process-image", mock.MatchedBy(func(v MaskingRequest) bool {
			capturedRequest = v
			return true
		})).
		Return(MaskingResponse{JobID: "new-job-id"}, nil)

	image := readTestImage(t)
	masking := PrepareForMasking(image)

	response, err := testClient.RequestMasking(masking)

	require.NoError(t, err)
	assert.Equal(t, "new-job-id", response.JobID)
	assert.True(t, bool(capturedRequest.Humans), "Humans detection should be enabled by default")
}

func TestRequestMasking_Validation(t *testing.T) {
	tests := []struct {
		name          string
		shape         Shape
		expectedError string
	}{
		{
			name:          "Invalid shape returns error",
			shape:         "invalid",
			expectedError: "invalid shape: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewMockTransport(t)
			testClient := createTestClient(transport)
			image := readTestImage(t)

			masking := PrepareForMasking(image)
			masking.Shape = tt.shape

			_, err := testClient.RequestMasking(masking)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func readTestImage(t *testing.T) io.Reader {
	t.Helper()

	file, err := os.Open("testdata/damonstration.jpg")
	require.NoError(t, err, "Failed to open test image")

	t.Cleanup(func() {
		_ = file.Close()
	})

	return file
}

func createTestClient(transport *MockTransport) *MaskingService {
	return NewMaskingService(WithTransport(transport))
}

func TestGetJobStatus(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		GetJobStatus("https://app.maskit.ai/api/v1/masking/image-status?jobid=job-1").
		Return(ImageStatusResponse{JobID: "job-1", Status: JobStatusReadyToDownload}, nil)

	resp, err := client.GetJobStatus("job-1")

	require.NoError(t, err)
	assert.Equal(t, "job-1", resp.JobID)
	assert.Equal(t, JobStatusReadyToDownload, resp.Status)
}

func TestDownloadImage(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)
	expectedData := io.NopCloser(strings.NewReader("fake-image-data"))

	transport.EXPECT().
		DownloadImage("https://app.maskit.ai/api/v1/masking/image-download?jobid=job-1").
		Return(expectedData, nil)

	reader, err := client.DownloadImage("job-1")

	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "fake-image-data", string(data))
}

func TestMaskImage_FullFlow(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.AnythingOfType("maskit.MaskingRequest")).
		Return(MaskingResponse{JobID: "job-42"}, nil)

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-42", Status: JobStatusInProgress}, nil).
		Once()

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-42", Status: JobStatusReadyToDownload}, nil).
		Once()

	transport.EXPECT().
		DownloadImage(mock.AnythingOfType("string")).
		Return(io.NopCloser(strings.NewReader("masked-image-data")), nil)

	ctx := context.Background()
	data, err := client.MaskImage(ctx, strings.NewReader("test-image"))

	require.NoError(t, err)
	assert.Equal(t, "masked-image-data", string(data))
}

func TestMaskImage_CancelledContext(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.AnythingOfType("maskit.MaskingRequest")).
		Return(MaskingResponse{JobID: "job-cancel"}, nil)

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-cancel", Status: JobStatusInProgress}, nil).
		Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.MaskImage(ctx, strings.NewReader("test-image"))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMaskImage_JobFailed(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.AnythingOfType("maskit.MaskingRequest")).
		Return(MaskingResponse{JobID: "job-fail"}, nil)

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-fail", Status: JobStatusFailed}, nil)

	ctx := context.Background()
	_, err := client.MaskImage(ctx, strings.NewReader("test-image"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestMaskImage_WithOptions(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	var captured MaskingRequest
	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.MatchedBy(func(r MaskingRequest) bool {
			captured = r
			return true
		})).
		Return(MaskingResponse{JobID: "job-opt"}, nil)

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-opt", Status: JobStatusReadyToDownload}, nil)

	transport.EXPECT().
		DownloadImage(mock.AnythingOfType("string")).
		Return(io.NopCloser(strings.NewReader("data")), nil)

	ctx := context.Background()
	data, err := client.MaskImage(ctx,
		strings.NewReader("test-image"),
		WithMethod(MethodBlackFill),
		WithShape(ShapeRectangle),
		WithFaces(false),
		WithBlurStrength(50),
	)

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Equal(t, MethodBlackFill, captured.Method)
	assert.Equal(t, ShapeRectangle, captured.Shape)
	assert.False(t, bool(captured.Faces))
	assert.Equal(t, 50, captured.BlurStrength)
}

func TestMaskImage_JobTimedOut(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.AnythingOfType("maskit.MaskingRequest")).
		Return(MaskingResponse{JobID: "job-timeout"}, nil)

	transport.EXPECT().
		GetJobStatus(mock.AnythingOfType("string")).
		Return(ImageStatusResponse{JobID: "job-timeout", Status: JobStatusTimedOut}, nil)

	ctx := context.Background()
	_, err := client.MaskImage(ctx, strings.NewReader("test-image"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestMaskImage_SendError(t *testing.T) {
	transport := NewMockTransport(t)
	client := createTestClient(transport)

	transport.EXPECT().
		Send(mock.AnythingOfType("string"), mock.AnythingOfType("maskit.MaskingRequest")).
		Return(MaskingResponse{}, assert.AnError)

	ctx := context.Background()
	_, err := client.MaskImage(ctx, strings.NewReader("test-image"))
	assert.ErrorIs(t, err, assert.AnError)
}
