package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	exitGeneral  = 1
	exitAuth     = 2
	exitNotFound = 3
)

// handleAPIError prints a user-friendly error from an API response and exits.
func handleAPIError(status int, body []byte) {
	msg := parseErrorMessage(body)

	switch {
	case status == 401 || status == 403:
		fmt.Fprintf(os.Stderr, "Error: %s\nRun 'arsenale login' to authenticate.\n", msg)
		os.Exit(exitAuth)
	case status == 404:
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(exitNotFound)
	default:
		if verbose {
			fmt.Fprintf(os.Stderr, "Error (HTTP %d): %s\n", status, msg)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		}
		os.Exit(exitGeneral)
	}
}

// checkAPIError checks the status code and handles errors if needed.
// Returns true if there was an error (and the program has exited or should exit).
func checkAPIError(status int, body []byte) {
	if status >= 200 && status < 300 {
		return
	}
	handleAPIError(status, body)
}

// parseErrorMessage extracts an error message from an API JSON response.
func parseErrorMessage(body []byte) string {
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return errResp.Error
		}
		if errResp.Message != "" {
			return errResp.Message
		}
	}
	if len(body) > 0 {
		return string(body)
	}
	return "unknown error"
}

// fatal prints an error message and exits.
func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(exitGeneral)
}

// fatalAuth prints an auth error and exits.
func fatalAuth(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(exitAuth)
}
