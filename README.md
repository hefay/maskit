[![Tests](https://github.com/hefay/maskit/actions/workflows/tests.yml/badge.svg)](https://github.com/hefay/maskit/actions/workflows/tests.yml)

### MaskIt Go SDK

A Go client library for interacting with the [MaskIt API](https://www.maskit.ai/). This package allows you to easily integrate automated anonymization of faces, humans, and license plates into your Go applications.

---

## Features

* **Automated Detection**: Support for faces, full human bodies, and license plates.
* **Flexible Masking Shapes**: Choose between soft polygons (`ShapeMask`) or standard bounding boxes (`ShapeRectangle`).
* **Anonymization Methods**: Support for Gaussian blur (`Blur`) or solid color fills (`BlackFill`).
* **Production Ready**: Built-in support for custom transports and easy integration with existing `io.Reader` streams.

---

## Installation

```bash
go get github.com/yourusername/maskit-go

```

## Quick Start

To use the MaskIt API, you will need an API Key from the [MaskIt Dashboard](https://www.google.com/search?q=https://app.maskit.ai/).

```go
package main

import (
	"fmt"
	"os"
	"github.com/yourusername/maskit-go"
)

func main() {
	// Open an image to be masked
	file, err := os.Open("input.jpg")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Initialize the service
	service := maskit.NewMaskingService()

	// Prepare a default request 
	// (Masks faces, humans, and plates using Blur by default)
	req := maskit.PrepareForMasking(file)
	
	// Customize the request if needed
	req.Method = maskit.MethodBlackFill
	req.BlurStrength = 50

	// Send the request to MaskIt API
	resp, err := service.RequestMasking(req)
	if err != nil {
		fmt.Printf("Error during masking: %v\n", err)
		return
	}

	fmt.Printf("Request successfully submitted. Job ID: %s\n", resp.JobID)
}

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
| `UseWebhook` | `bool` | Whether to receive results via a configured webhook. |

---

## Advanced Usage

### Custom Transport

If you need to customize the underlying HTTP client (e.g., adding custom timeouts, logging, or middleware), you can implement the `Transport` interface and pass it using functional options:

```go
service := maskit.NewMaskingService(
    maskit.WithTransport(myCustomTransport),
)

```

### Shape Validation

The SDK includes a helper to ensure your requested masking shape is supported before the API call is made:

```go
if !req.Shape.IsValid() {
    // Handle invalid shape
}

```

---

## Documentation

For more detailed information about the underlying API, visit the [official MaskIt Documentation](https://docs.maskit.ai/).

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

---

**Would you like me to add a section on how to handle the API response or implement a custom HTTP transport with API key headers?**
