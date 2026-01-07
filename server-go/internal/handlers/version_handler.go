package handlers

import (
	"encoding/json"
	"net/http"
)

// Version information injected at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VersionResponse{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	})
}
