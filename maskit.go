// Package maskit provides tools for interacting with the MaskIt API.
package maskit

import (
	"fmt"
	"io"
)

const (
	// MaskitAPIURL is the base URL for the MaskIt API.
	MaskitAPIURL = "https://app.maskit.ai/api/v1"
	// APIKeyHeader is the header key used for authentication.
	APIKeyHeader = "X-Api-Key"
)

// Humans indicates whether to detect and mask humans in the image.
type Humans bool

// Faces indicates whether to detect and mask faces in the image.
type Faces bool

// LicensePlates indicates whether to detect and mask license plates in the image.
type LicensePlates bool

// Shape used to mask areas: [ShapeMask] (soft polygon) or [ShapeRectangle] (bounding box).
type Shape string

//goland:noinspection GoUnusedConst
const (
	// ShapeMask uses a soft polygon for masking
	ShapeMask Shape = `Mask`
	// ShapeRectangle uses a bounding box for masking
	ShapeRectangle Shape = `Rectangle`
)

func (s Shape) IsValid() bool {
	switch s {
	case ShapeRectangle, ShapeMask:
		return true
	}
	return false
}

// Method is a method used for masking.
type Method string

//goland:noinspection GoUnusedConst
const (
	// MethodBlur indicates that a blur will be used for masking.
	MethodBlur Method = `Blur`
	// MethodBlackFill indicates that a black fill will be used for masking.
	MethodBlackFill Method = `BlackFill`
)

// MaskingRequest represents a request sent to the masking API.
type MaskingRequest struct {
	Image io.Reader
	// LicensePlates indicates whether to detect and mask license plates in the image.
	Faces Faces
	// Humans indicates whether to detect and mask humans in the image.
	Humans Humans
	// LicensePlates indicates whether to detect and mask license plates in the image.
	LicensePlates LicensePlates
	// Shape used to mask areas: [ShapeMask] (soft polygon) or [ShapeRectangle] (bounding box).
	Shape        Shape
	Method       Method
	BlurStrength int
	EdgeBlurSize float32
	Metadata     string
	UseWebhook   bool
}

// MaskingResponse represents the response from a masking request.
type MaskingResponse struct {
	JobID string `json:"JobId"`
}

//go:generate mockery
type Transport interface {
	Send(string, MaskingRequest) (MaskingResponse, error)
}

type MaskingService struct {
	transport Transport
	apiKey string
}

type MaskingServiceOption func(*MaskingService)

func WithTransport(transport Transport) MaskingServiceOption {
	return func(s *MaskingService) {
		s.transport = transport
	}
}

func WithApiKey(key string) MaskingServiceOption {
	return func(s *MaskingService) {
		s.apiKey = key
	}
}

// NewMaskingService creates a new instance of MaskingService.
func NewMaskingService(options ...MaskingServiceOption) *MaskingService {
	transport := &HTTPTransport{}
	service := &MaskingService{
		transport: transport,
	}

	for _, opt := range options {
		opt(service)
	}

	return service
}

func (ms *MaskingService) RequestMasking(req MaskingRequest) (MaskingResponse, error) {
	if !req.Shape.IsValid() {
		return MaskingResponse{}, fmt.Errorf("invalid shape: %s", req.Shape)
	}
	response, err := ms.transport.Send(MaskitAPIURL+"/masking/process-image", req)
	if err != nil {
		return MaskingResponse{}, err
	}
	return response, nil
}

// PrepareForMasking creates a default MaskingRequest for the given image.
func PrepareForMasking(image io.Reader) MaskingRequest {
	return MaskingRequest{
		Image:         image,
		Faces:         true,
		Humans:        true,
		LicensePlates: true,
		Shape:         ShapeMask,
		Method:        MethodBlur,
		BlurStrength:  30,
		EdgeBlurSize:  0.2,
	}
}
