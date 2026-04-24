package main

import (
	"fmt"
	"net/http"
	"time"

	internal "github.com/thomseddon/traefik-forward-auth/internal"
)

// Main
func main() {
	// Parse options
	config := internal.NewGlobalConfig()

	// Setup logger
	log := internal.NewDefaultLogger()

	// Perform config validation
	config.Validate()

	// Build server
	server := internal.NewServer()

	// Attach router to default server
	http.HandleFunc("/", server.RootHandler)

	// ReadHeaderTimeout caps the time to read the request line + headers, which
	// is the slow-loris / resource-exhaustion mitigation gosec G114 flagged.
	// Read/Write timeouts cover the whole request cycle including the upstream
	// OIDC token/userinfo round-trip in the callback path. IdleTimeout bounds
	// keep-alive connections.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start
	log.WithField("config", config).Debug("Starting with config")
	log.Infof("Listening on :%d", config.Port)
	log.Info(srv.ListenAndServe())
}
