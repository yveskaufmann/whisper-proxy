package transcribe

type AudioTranscriber interface {
	ProcessAudio(samples []float32) ([]string, error)
	Close() error
}
