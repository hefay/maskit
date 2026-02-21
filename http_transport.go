package maskit

// HTTPTransport handles the low-level communication.
type HTTPTransport struct {
	// Serializer a Deserializer by měly mít definované metody,
	// jinak je lepší je zatím vynechat.
}

func (*HTTPTransport) Send(url string, payload MaskingRequest) (MaskingResponse, error) {
	return MaskingResponse{}, nil
}
