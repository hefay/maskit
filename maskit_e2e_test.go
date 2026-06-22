package maskit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaskImage_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	key := os.Getenv("MASKIT_API_KEY")
	if key == "" {
		t.Skip("MASKIT_API_KEY not set, skipping e2e test")
	}

	svc := NewMaskingService(
		WithApiKey(key),
	)

	f, err := os.Open("testdata/damonstration.jpg")
	require.NoError(t, err)
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	data, err := svc.MaskImage(ctx, f)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	out := filepath.Join(t.TempDir(), "masked.jpg")
	require.NoError(t, os.WriteFile(out, data, 0644))
	t.Logf("masked image saved to %s", out)
}
