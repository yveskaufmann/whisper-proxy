package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-audio/wav"
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

		waveFile, err := convertWebmToWav(file); 
		if err != nil {
			log.Printf("FFmpeg conversion failed: %v", err)
			http.Error(w, "FFmpeg conversion failed", http.StatusInternalServerError)
			return
		}
		defer waveFile.Close()

		whisperResult, err := invokeTranscriptionService(w, waveFile, o)
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

func invokeTranscriptionService(w http.ResponseWriter, wavReader *os.File, o *Options) (*TranscriptionResult, error) {
	
	defer wavReader.Close()

	if o.AudioTranscriber != nil {
		decoder := wav.NewDecoder(wavReader)
		if !decoder.IsValidFile() {
			return nil, fmt.Errorf("invalid WAV file")
		}

		bitDepth := decoder.BitDepth
		channels := decoder.NumChans
		
		log.Printf("WAV file info: BitDepth=%d, Channels=%d, len=%d", bitDepth, channels, decoder.PCMLen())
		
		if channels != 1 {
			return nil, fmt.Errorf("only mono WAV files are supported, but got %d channels", channels)
		}

		buffer, err := decoder.FullPCMBuffer()
		if err != nil {
			return nil, err
		}

		if len(buffer.Data) == 0 {
			log.Printf("WARNING: audio file conversation results into 0 samples, returning empty transcription result")
			return &TranscriptionResult{
				Text:       "",
				StatusCode: http.StatusOK,
			}, nil
		}

		samples := make([]float32, len(buffer.Data))
		for i, sample := range buffer.Data {
			samples[i] = float32(sample) / float32(int(1)<<(bitDepth-1))
		}

		segments, err := o.AudioTranscriber.ProcessAudio(samples)
		if err != nil {
			return nil, fmt.Errorf("failed to process audio: %w", err)
		}

		text := strings.Join(segments, "")
		log.Printf("Received raw transcribe response: %s", text)

		return &TranscriptionResult{
			Text:       text,
			StatusCode: http.StatusOK,
		}, nil	
	}
	
	inferenceURL, err := url.JoinPath(o.WhisperURL, "inference")
	if err != nil {
		return nil, fmt.Errorf("Failed to create inference URL [baseURL=%q path=%q]: %w", o.WhisperURL, "inference", err)
	}
	
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() { 
		defer pw.Close()
		defer writer.Close()

		filePart, err := writer.CreateFormFile("file", "audio.wav")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("Failed to create multipart form file entry: %w", err))
			return
		}

		if _, err := io.Copy(filePart, wavReader); err != nil {
			pw.CloseWithError(fmt.Errorf("Failed to copy audio to multipart form file: %w", err))
			return
		}

		if err := writer.WriteField("response-format", "json"); err != nil {
			pw.CloseWithError(fmt.Errorf("Failed to set form field response-format: %w", err))
			return
		}
	}()


	req, err := http.NewRequest("POST", inferenceURL, pr)
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

