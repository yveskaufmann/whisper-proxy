package proxy

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

// convertWebmToWav converts a WebM audio file to WAV format using ffmpeg.
func convertWebmToWav(src io.Reader) (file *os.File, err error) {  

	waveFile, err := os.CreateTemp("", "out-*.wav")
	if err != nil {
		return nil, fmt.Errorf("Failed to create temp file: %w", err)
	}
	waveFile.Close()

	log.Println("Translating audio format via ffmpeg to 16kHz WAV...")

	cmd := exec.Command("ffmpeg",
		"-fflags", "+discardcorrupt",
		"-err_detect", "ignore_err",
		"-f", "webm",
		"-i", "-",
		"-y",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		waveFile.Name(),		
	)

	cmd.Stdin = src

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	file, err = os.Open(waveFile.Name())
	if err != nil {
		return nil, fmt.Errorf("Failed to open converted WAV file: %w", err)
	}

	return file, nil
}

