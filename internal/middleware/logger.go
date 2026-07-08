package middleware

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"time"
)

type recordingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	wroteHeader bool
}

func newRecordingResponseWriter(w http.ResponseWriter) *recordingResponseWriter {
	return &recordingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code is 200 OK
		wroteHeader:    false,
	}
}

func (rw *recordingResponseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *recordingResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer only if it actually supports it,
// so streaming handlers (SSE) keep working through this wrapper.
func (rw *recordingResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack forwards to the underlying writer only if it actually supports it,
// so websocket upgrade handlers keep working through this wrapper.
func (rw *recordingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}


func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("-> http %s %s", r.Method, r.URL.Path)
		rw := newRecordingResponseWriter(w)

		startTime := time.Now()
		next.ServeHTTP(rw, r)
		duration := time.Since(startTime)

		if !rw.wroteHeader {
			rw.statusCode = -1
			log.Printf("Warning: ResponseWriter did not write a header for %s %s", r.Method, r.URL.Path)
		}

		log.Printf("<- http %s %s %d (%d ms)", r.Method, r.URL.Path, rw.statusCode, duration.Milliseconds())
	})
}
