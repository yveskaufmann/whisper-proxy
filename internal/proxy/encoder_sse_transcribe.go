package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type sseEncoderTranscribe struct{}

func (e *sseEncoderTranscribe) Encode(w http.ResponseWriter, result *TranscriptionResult) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	
	event := struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} {
		Type: "transcript.text.done",
		Text: result.Text,
	}

	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonBytes); err != nil {
		return err
	}
	
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}