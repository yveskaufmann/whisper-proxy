package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"com.yv35.whisper-server-proxy/internal/middleware"
	"com.yv35.whisper-server-proxy/internal/transcribe"
)

type Options struct {
	// Port is the port on which the proxy will listen for incoming requests.
	Port int `json:"port"`

	// BackendURL is the URL of the backend service to which the proxy will forward requests.
	WhisperURL string `json:"whisper_url"`

	// The transcriber implementation to use for processing audio. If not set, the the whisper-server will be used.
	AudioTranscriber transcribe.AudioTranscriber	 

}

type Proxy struct {
	Options *Options
	server  *http.Server
}

func NewProxy(options *Options) *Proxy {
	return &Proxy{
		Options: options,
	}
}

func (p *Proxy) Start() error {

	mux := http.NewServeMux()
	mux.HandleFunc("/health", newHealthCheckHandler(p.Options))
	mux.HandleFunc("/audio/transcriptions", newTranscribeHandler(p.Options))

	// Create a new HTTP server with the specified options
	p.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", p.Options.Port),
		Handler:      middleware.Logger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start the server in a separate goroutine
	go func() {
		log.Printf("Server is running on http://localhost:%d", p.Options.Port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start http server %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop // Block here until a signal is received

	log.Println("Shutting down server safely...")

	// Give active connections 5 seconds to finish their requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped.")

	return nil
}

func (p *Proxy) Stop() error {

	if p.server != nil {
		log.Printf("Stopping server on http://localhost:%d", p.Options.Port)
		return p.server.Close()
	}

	if p.Options.AudioTranscriber != nil {
		if err := p.Options.AudioTranscriber.Close(); err != nil {
			log.Printf("Failed to stop AudioTranscriber: %v ", err)
		}
		p.Options.AudioTranscriber = nil
	}

	return nil
}
