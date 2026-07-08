package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

const MAX_MULTIPART_MEMORY = 10 << 20 // 10 MB

func newTranscribeHandler(o *Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseMultipartForm(MAX_MULTIPART_MEMORY)
		if err != nil {
			log.Printf("Failed to parse multipart form: %v", err)
			http.Error(w, "Failed to parse multipart form file too large", http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			log.Printf("Failed to get file from form: %v", err)
			http.Error(w, "Failed to get file from form", http.StatusBadRequest)
			return
		}
		defer file.Close()
		
		stream := r.FormValue("stream")
		if stream == "" {
			stream = "false"
		}

		log.Printf("Received transcription request with stream: %s, file: %s", stream, fileHeader.Filename)

		// for key, values := range r.Form {
		// 	log.Printf("Form field: %s, Values: %v", key, values)
		// }

		// for key, values := range r.Header {
		// 	log.Printf("Header field: %s = %v", key, values)
		// }

		var originalBytes bytes.Buffer
		if _, err := io.Copy(&originalBytes, file); err != nil {
			log.Printf("Error reading original file bytes: %v", err)
			http.Error(w, "Failed to read original file data", http.StatusInternalServerError)
			return
		}

		log.Println("Translating audio format via ffmpeg to 16kHz WAV...")
		wavBytes, err := convertWebmToWav(originalBytes.Bytes())
		if err != nil {
			log.Printf("FFmpeg conversion failed: %v", err)
			http.Error(w, "Failed to decode/convert audio data", http.StatusInternalServerError)
			return
		}

		whisperResult, err := invokeTranscriptionService(w, wavBytes, o)
		if err != nil {
			log.Printf("failed to invoke transcription service: %v", err)

			statusCode := http.StatusInternalServerError
			if whisperResult != nil && whisperResult.StatusCode != 0 {
				statusCode = whisperResult.StatusCode
			}

			http.Error(w, "failed to invoke transcription service", statusCode)
			return
		}

		encoder := encoderFor(stream == "true")
		if err := encoder.Encode(w, &TranscriptionResult{
			Text:       whisperResult.Text,
			StatusCode: whisperResult.StatusCode,
		}); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func invokeTranscriptionService(w http.ResponseWriter, wavBytes []byte, o *Options) (*TranscriptionResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("Failed to create multipart form file entry: %w", err)		
	}

	if _, err := io.Copy(part, bytes.NewReader(wavBytes)); err != nil {
		return nil, fmt.Errorf("Failed to copy audio to multipart form file: %w", err)
	}

	if err := writer.WriteField("response-format", "json"); err != nil {
		return nil, fmt.Errorf("Failed to set form field response-format: %w", err)
	}
	writer.Close()

	inferenceURL, err := url.JoinPath(o.WhisperURL, "inference")
	if err != nil {
		return nil, fmt.Errorf("Failed to create inference URL [baseURL=%q path=%q]: %w", o.WhisperURL, "inference", err)
	}

	req, err := http.NewRequest("POST", inferenceURL, body)
	if err != nil {
		return nil, fmt.Errorf("Failed to create transcribe request to backend: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to forward transcribe request to backend: %w", err)
	}
	defer resp.Body.Close()

	result := &TranscriptionResult{
		StatusCode: resp.StatusCode,
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("Failed to read transcribe response from backend: %w", err)
	}

	log.Printf("Received raw transcribe response: %s", respBody)

	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, fmt.Errorf("Failed to JSON unmarshal transcribe response from backend: %w", err)
	}
	return result, nil
}

