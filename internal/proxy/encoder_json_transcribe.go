package proxy

import (
	"encoding/json"
	"net/http"
)

type jsonEncoderTranscribe struct{}

func (e *jsonEncoderTranscribe) Encode(w http.ResponseWriter, result *TranscriptionResult) error {
	openAIResponse := map[string]string{ 
		"text": result.Text,
	}

	finalJSON, err := json.Marshal(openAIResponse)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(result.StatusCode)
	w.Write(finalJSON)

	return nil
}