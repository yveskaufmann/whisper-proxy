package transcribe

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)


type whisperTranscriber struct {
	lock sync.Mutex
	modelPath string
	model whisper.Model
	
	lastUsedAt time.Time
	modelKeepAliveDuration time.Duration
	keepAliveTicker *time.Ticker
	stopChan chan bool
}

type Option func(*whisperTranscriber)

func WithModelKeepAliveDuration(duration time.Duration) Option {
	return func(wt *whisperTranscriber) {
		wt.modelKeepAliveDuration = duration
	}
}

func NewWhisperTranscriber(path string, opts ...Option) (*whisperTranscriber, error) {

	// Only expand $HOME environment variable to avoid sensitive variable exposure e.g API_KEY
	path = os.Expand(path, func(key string) string{
		if key == "HOME" {
			home, _ := os.UserHomeDir()
			return home
		}
		return "$" + key
	})
	

	instance :=  &whisperTranscriber{
		modelPath: path,
		modelKeepAliveDuration: 5 * time.Minute,
		stopChan: make(chan bool),
	}

	for _, opt := range opts {
		opt(instance)
	}

	if instance.modelKeepAliveDuration > 0 {
		instance.keepAliveTicker = time.NewTicker(instance.modelKeepAliveDuration)
		go func() {
			for  {
				select {
				case <-instance.keepAliveTicker.C:
					if instance.model == nil {
						continue
					}
					instance.lock.Lock()
					if instance.model != nil && time.Since(instance.lastUsedAt) > instance.modelKeepAliveDuration {
						log.Printf("Whisper model %q has not been used for %v, closing it to free resources\n", instance.modelPath, instance.modelKeepAliveDuration)
						if err := instance.model.Close(); err != nil {
							log.Printf("Failed to close whisper model %q: %v\n", instance.modelPath, err)
						}
						instance.model = nil
					}
					instance.lock.Unlock()
				case <-instance.stopChan:
					log.Printf("Stopping keep-alive ticker for whisper model %q\n", instance.modelPath)
					return
				}
			}	
		}()
	}

	if err := instance.loadModel(path); err != nil {
		return nil, err
	}

	return instance, nil	
}

func (wt* whisperTranscriber) loadModel(path string) (error) {
	if wt.model != nil {
		if err := wt.model.Close(); err != nil {
			return fmt.Errorf("failed to close previous whisper model %q: %w", wt.modelPath, err)
		}
		log.Printf("Closed previous whisper model %q\n", wt.modelPath)
	}

	model, err := whisper.New(path)
	if err != nil {
		return fmt.Errorf("failed to load whisper model %q: %w", path, err)
	}
	wt.model = model
	wt.modelPath = path
	wt.lastUsedAt = time.Now()	

	log.Printf("Loaded whisper model %q\n", path)

	return nil
}

func (wt* whisperTranscriber) ProcessAudio(samples []float32) ([]string, error) {

	wt.lock.Lock()
	defer wt.lock.Unlock()

	if wt.model == nil {
		if err := wt.loadModel(wt.modelPath); err != nil {
			return nil, fmt.Errorf("failed to load whisper model %q: %w", wt.modelPath, err)
		}
	}

	wt.lastUsedAt = time.Now()
	context, err := wt.model.NewContext()
	context.SetLanguage("auto")

	if err != nil {
		return nil, fmt.Errorf("failed to create whisper context: %w", err)
	}

	if err := context.Process(samples, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process samples: %w", err)
	}

	var segments []string
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break
		}
		segments = append(segments, segment.Text)
	}

	return segments, nil

}

func (wt* whisperTranscriber) Close() (error) {
	if wt.model == nil {
		return nil
	}

	wt.lock.Lock()
	defer wt.lock.Unlock()

	wt.keepAliveTicker.Stop()
	wt.stopChan <- true

	if wt.model != nil {
		if err := wt.model.Close(); err != nil {
			return fmt.Errorf("failed to close whisper model %q: %w", wt.modelPath, err)
		}
		wt.model = nil
	}
	return nil
}

