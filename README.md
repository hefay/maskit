[![Tests](https://github.com/hefay/maskit/actions/workflows/tests.yml/badge.svg)](https://github.com/hefay/maskit/actions/workflows/tests.yml)

### MaskIt Go SDK

A Go client library for interacting with the [MaskIt API](https://www.maskit.ai/). This package allows you to easily integrate automated anonymization of faces, humans, and license plates into your Go applications.

---

## Features

* **Automated Detection**: Support for faces, full human bodies, and license plates.
* **Flexible Masking Shapes**: Choose between soft polygons (`ShapeMask`) or standard bounding boxes (`ShapeRectangle`).
* **Anonymization Methods**: Support for Gaussian blur (`Blur`) or solid color fills (`BlackFill`).
* **High-Level API**: `MaskImage` polls and downloads the result in one call.
* **Low-Level API**: `RequestMasking`, `GetJobStatus`, `DownloadImage` for full control.
* **Context Support**: All blocking operations respect `context.Context` for cancellation and timeouts.
* **Production Ready**: Built-in support for custom transports and easy integration with existing `io.Reader` streams.

---

## Installation

```bash
go get github.com/hefay/maskit

```

## Quick Start

To use the MaskIt API, you will need an API Key from the [MaskIt Dashboard](https://www.google.com/search?q=https://app.maskit.ai/).

### High-Level API (recommended)

Submit an image, poll for completion, and download the result — all in one call:

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/hefay/maskit"
)

func main() {
	apiKey := os.Getenv("MASKIT_API_KEY")

	file, err := os.Open("input.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	service := maskit.NewMaskingService(maskit.WithApiKey(apiKey))

	ctx := context.Background()
	data, err := service.MaskImage(ctx, file,
		maskit.WithMethod(maskit.MethodBlackFill),
		maskit.WithBlurStrength(50),
	)
	if err != nil {
		log.Fatal(err)
	}

	os.WriteFile("output_masked.jpg", data, 0644)
	log.Println("Masked image saved to output_masked.jpg")
}
```

### Low-Level API

For full control over each step:

```go
ctx := context.Background()
apiKey := os.Getenv("MASKIT_API_KEY")
service := maskit.NewMaskingService(maskit.WithApiKey(apiKey))

// 1. Submit the image
req := maskit.PrepareForMasking(file)
resp, err := service.RequestMasking(ctx, req)
// ...

// 2. Poll for status (or use GetJobStatus directly)
status, err := service.GetJobStatus(ctx, resp.JobID)
// ...

// 3. Download the result
reader, err := service.DownloadImage(ctx, resp.JobID)
defer reader.Close()
data, _ := io.ReadAll(reader)
```

---

## Configuration: `MaskingRequest`

| Field | Type | Description |
| --- | --- | --- |
| `Image` | `io.Reader` | The image data stream. |
| `Faces` | `bool` | Detect and mask faces. |
| `Humans` | `bool` | Detect and mask entire human figures. |
| `LicensePlates` | `bool` | Detect and mask vehicle license plates. |
| `Shape` | `Shape` | Mask shape: `ShapeMask` (soft polygon) or `ShapeRectangle` (box). |
| `Method` | `Method` | Anonymization type: `MethodBlur` or `MethodBlackFill`. |
| `BlurStrength` | `int` | Intensity of the blur effect. |
| `EdgeBlurSize` | `float32` | Softness of the mask edges. |
| `Metadata` | `string` | Optional metadata (e.g. user id, context). |
| `UseWebhook` | `bool` | Whether to receive results via a configured webhook. |

### `MaskOption` Functions

Use with `MaskImage` to override defaults:

- `WithShape(Shape)`
- `WithMethod(Method)`
- `WithFaces(bool)`
- `WithHumans(bool)`
- `WithLicensePlates(bool)`
- `WithBlurStrength(int)`
- `WithEdgeBlurSize(float32)`
- `WithMetadata(string)`
- `WithWebhook(bool)`

---

## Advanced Usage

### Custom Transport

Implement the `Transport` interface for custom HTTP clients, logging, or middleware:

```go
service := maskit.NewMaskingService(
    maskit.WithTransport(myCustomTransport),
)

```

### Job Status Values

The SDK defines the following `JobStatus` constants:

| Constant | Description |
| --- | --- |
| `JobStatusPending` | Job is queued but not yet started |
| `JobStatusInProgress` | Job is currently being processed |
| `JobStatusReadyToDownload` | Image is processed and ready for download |
| `JobStatusCompleted` | Job completed and post-processing succeeded |
| `JobStatusTimedOut` | Job did not complete within the allowed time |
| `JobStatusFailed` | An error occurred during processing |

---

## Documentation

For more detailed information about the underlying API, visit the [official MaskIt Documentation](https://docs.maskit.ai/).

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
