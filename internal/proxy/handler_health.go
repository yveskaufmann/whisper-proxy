package proxy

import (
	"encoding/json"
	"net/http"
)

func newHealthCheckHandler(o *Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Status string `json:"status"`
		}{
			Status: "OK",
		}

		w.Header().Set("Content-Type", "application/json")
		jsonData, _ := json.Marshal(response)

		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}
}
