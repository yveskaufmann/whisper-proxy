package proxy

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// convertWebmToWav converts a WebM audio file to WAV format using ffmpeg.
func convertWebmToWav(webmBytes []byte) ([]byte, error) {
	// 1. Prepare input and output temporary files

	// The WAV muxer needs to seek back after writing the data chunk to patch
	// the RIFF/data chunk size fields. pipe:1 is non-seekable, so those sizes
	// end up as placeholder/unknown-length values, which some WAV readers
	// (including whisper.cpp) reject. Write to a real file instead.

	inFile, errIn := os.CreateTemp("", "whsipr-input-*.webm")
	outFile, errOut := os.CreateTemp("", "whsipr-output-*.wav")

	defer closeAndRemoveFile(inFile)
	defer closeAndRemoveFile(outFile)

	if errIn != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", errIn)
	}

	if errOut != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", errOut)
	}

	if _, err := inFile.Write(webmBytes); err != nil {
		return nil, fmt.Errorf("failed to write temp input file: %w", err)
	}

	if err := inFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp input file: %w", err)
	}

	if err := outFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp output file: %w", err)
	}

	// 2. Run ffmpeg to convert the WebM file to WAV format

	cmd := exec.Command("ffmpeg",
		"-y",
		"-err_detect", "ignore_err",
		"-fflags", "+discardcorrupt",
		"-i", inFile.Name(),
		"-f", "wav",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		outFile.Name(),
	)

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return os.ReadFile(outFile.Name())
}

func closeAndRemoveFile(f *os.File) {
	if f != nil {
		err := f.Close()
		if err != nil && !errors.Is(err, os.ErrClosed) {
			log.Printf("Failed to close file: %v\n", err)
		}

		if err := os.Remove(f.Name()); err != nil {
			log.Printf("Failed to remove file: %v\n", err)
		}
	}
}
