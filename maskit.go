package maskit

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	MaskitAPIURL      = "https://app.maskit.ai/api/v1"
	APIKeyHeader      = "X-Api-Key"
	pathProcessImage  = "/masking/process-image"
	pathImageStatus   = "/masking/image-status"
	pathImageDownload = "/masking/image-download"
	pollInterval      = 2 * time.Second
)

type Humans bool
type Faces bool
type LicensePlates bool

type Shape string

const (
	ShapeMask      Shape = `Mask`
	ShapeRectangle Shape = `Rectangle`
)

func (s Shape) IsValid() bool {
	switch s {
	case ShapeRectangle, ShapeMask:
		return true
	}
	return false
}

type Method string

const (
	MethodBlur      Method = `Blur`
	MethodBlackFill Method = `BlackFill`
)

type JobStatus string

const (
	JobStatusPending        JobStatus = "Pending"
	JobStatusInProgress     JobStatus = "InProgress"
	JobStatusReadyToDownload JobStatus = "ReadyToDownload"
	JobStatusCompleted      JobStatus = "Completed"
	JobStatusTimedOut       JobStatus = "TimedOut"
	JobStatusFailed         JobStatus = "Failed"
)

type MaskingRequest struct {
	Image         io.Reader
	Faces         Faces
	Humans        Humans
	LicensePlates LicensePlates
	Shape         Shape
	Method        Method
	BlurStrength  int
	EdgeBlurSize  float32
	Metadata      string
	UseWebhook    bool
}

type MaskingResponse struct {
	JobID string `json:"JobId"`
}

type ImageStatusResponse struct {
	JobID  string    `json:"JobId"`
	Status JobStatus `json:"Status"`
}

//go:generate mockery
type Transport interface {
	Send(string, MaskingRequest) (MaskingResponse, error)
	GetJobStatus(string) (ImageStatusResponse, error)
	DownloadImage(string) (io.ReadCloser, error)
}

type MaskingService struct {
	transport Transport
	apiKey    string
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

func NewMaskingService(options ...MaskingServiceOption) *MaskingService {
	transport := &HTTPTransport{}
	service := &MaskingService{
		transport: transport,
	}

	for _, opt := range options {
		opt(service)
	}

	if t, ok := service.transport.(*HTTPTransport); ok && service.apiKey != "" {
		t.APIKey = service.apiKey
	}

	return service
}

func (ms *MaskingService) RequestMasking(req MaskingRequest) (MaskingResponse, error) {
	if !req.Shape.IsValid() {
		return MaskingResponse{}, fmt.Errorf("invalid shape: %s", req.Shape)
	}
	response, err := ms.transport.Send(MaskitAPIURL+pathProcessImage, req)
	if err != nil {
		return MaskingResponse{}, err
	}
	return response, nil
}

func (ms *MaskingService) GetJobStatus(jobID string) (ImageStatusResponse, error) {
	return ms.transport.GetJobStatus(MaskitAPIURL + pathImageStatus + "?jobid=" + jobID)
}

func (ms *MaskingService) DownloadImage(jobID string) (io.ReadCloser, error) {
	return ms.transport.DownloadImage(MaskitAPIURL + pathImageDownload + "?jobid=" + jobID)
}

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

type MaskOption func(*MaskingRequest)

func WithShape(s Shape) MaskOption {
	return func(r *MaskingRequest) { r.Shape = s }
}

func WithMethod(m Method) MaskOption {
	return func(r *MaskingRequest) { r.Method = m }
}

func WithFaces(v bool) MaskOption {
	return func(r *MaskingRequest) { r.Faces = Faces(v) }
}

func WithHumans(v bool) MaskOption {
	return func(r *MaskingRequest) { r.Humans = Humans(v) }
}

func WithLicensePlates(v bool) MaskOption {
	return func(r *MaskingRequest) { r.LicensePlates = LicensePlates(v) }
}

func WithBlurStrength(v int) MaskOption {
	return func(r *MaskingRequest) { r.BlurStrength = v }
}

func WithEdgeBlurSize(v float32) MaskOption {
	return func(r *MaskingRequest) { r.EdgeBlurSize = v }
}

func WithMetadata(v string) MaskOption {
	return func(r *MaskingRequest) { r.Metadata = v }
}

func WithWebhook(v bool) MaskOption {
	return func(r *MaskingRequest) { r.UseWebhook = v }
}

func (ms *MaskingService) waitForReady(ctx context.Context, jobID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := ms.GetJobStatus(jobID)
		if err != nil {
			return err
		}

		s := string(status.Status)
		switch {
		case strings.EqualFold(s, string(JobStatusReadyToDownload)),
			strings.EqualFold(s, string(JobStatusCompleted)):
			return nil
		case strings.EqualFold(s, string(JobStatusFailed)):
			return fmt.Errorf("job %s failed", jobID)
		case strings.EqualFold(s, string(JobStatusTimedOut)):
			return fmt.Errorf("job %s timed out", jobID)
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (ms *MaskingService) MaskImage(ctx context.Context, image io.Reader, opts ...MaskOption) ([]byte, error) {
	req := PrepareForMasking(image)
	for _, opt := range opts {
		opt(&req)
	}

	resp, err := ms.RequestMasking(req)
	if err != nil {
		return nil, err
	}

	if err := ms.waitForReady(ctx, resp.JobID); err != nil {
		return nil, err
	}

	reader, err := ms.DownloadImage(resp.JobID)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
