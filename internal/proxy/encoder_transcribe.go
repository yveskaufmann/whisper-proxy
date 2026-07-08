package proxy

import "net/http"

type TranscriptionResult struct {
	// Text is the transcribed text returned from the backend service.
	Text string `json:"text"`

	// StatusCode is the HTTP status code returned from the backend service.
	StatusCode int `json:"status_code,omitempty"`
}

// TranscribeResponseEncoder defines the interface for encoding a TranscribeResponse into an HTTP response.
type TranscribeResponseEncoder interface {
	Encode(responseWriter http.ResponseWriter, response *TranscriptionResult) error
}

func encoderFor(stream bool) TranscribeResponseEncoder {
	if stream {
		return &sseEncoderTranscribe{}
	}
	return &jsonEncoderTranscribe{}
}