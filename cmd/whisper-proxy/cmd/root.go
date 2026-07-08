package cmd

import (
	"fmt"
	"os"

	"com.yv35.whisper-server-proxy/internal/proxy"
	"github.com/spf13/cobra"
)

type Options struct {
	// Port is the port on which the proxy will listen for incoming requests.
	Port int `json:"port"`

	// BackendURL is the URL of the backend service to which the proxy will forward requests.
	WhisperURL string `json:"whisper_url"`

	// When set to true, the command will print the version information and exit.
	Version bool `json:"version"`
}

func CreateCommand() *cobra.Command {
	o := &Options{}
	cmd := &cobra.Command{
		Use:   "whisper-proxy",
		Short: "whisper-proxy is a proxy for whisper.cpp example server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startProxy(o)
		},
	}

	cmd.Flags().IntVar(&o.Port, "port", 8080, "Port on which the proxy will listen for incoming requests")
	cmd.Flags().StringVar(&o.WhisperURL, "whisper-url", "http://localhost:18080", "URL of the whisper.cpp backend service")
	cmd.Flags().BoolVar(&o.Version, "version", false, "Print the version information and exit")
	return cmd
}

func startProxy(o *Options) error {
	if o.Version {
		fmt.Println(FullVersionString())
		return nil
	}

	opts := &proxy.Options{
		Port:       o.Port,
		WhisperURL: o.WhisperURL,
	}

	p := proxy.NewProxy(opts)
	return p.Start()
}

func Execute() {
	cmd := CreateCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
