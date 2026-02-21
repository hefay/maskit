package maskit

import (
	"io"
	"os"
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
