// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	callbackPath     = "/callback"
	localServerPort  = "9090"
	callbackTimeout  = 5 * time.Minute
	shutdownTimeout  = 5 * time.Second
	successHTML      = `<html><body><h1>Authentication Successful!</h1><p>You can close this window and return to the CLI.</p></body></html>`
	errorHTML        = `<html><body><h1>Authentication Failed</h1><p>Error: %s</p><p>You can close this window and return to the CLI.</p></body></html>`
)

type oauthCallbackResult struct {
	code  string
	state string
	err   error
}

func performOAuthLogin(cmd *cobra.Command, provider string) error {
	ctx := cmd.Context()

	// Start local server to receive callback
	listener, err := net.Listen("tcp", "127.0.0.1:"+localServerPort)
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}

	callbackChan := make(chan oauthCallbackResult, 1)
	server := startCallbackServer(listener, callbackChan)

	// Get authorization URL from server with local callback URL
	callbackURL := fmt.Sprintf("http://127.0.0.1:%s%s", localServerPort, callbackPath)
	authURL, state, err := sdk.OAuthAuthorizationURL(ctx, provider, callbackURL)
	if err != nil {
		shutdownServer(server)
		return fmt.Errorf("failed to get authorization URL: %w", err)
	}

	// Open browser
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, please visit:\n%s\n\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	fmt.Println("Waiting for authentication callback...")

	// Wait for callback with timeout
	var result oauthCallbackResult
	select {
	case result = <-callbackChan:
		if result.err != nil {
			shutdownServer(server)
			return fmt.Errorf("callback error: %w", result.err)
		}
	case <-time.After(callbackTimeout):
		shutdownServer(server)
		return fmt.Errorf("authentication timeout after %v", callbackTimeout)
	}

	shutdownServer(server)

	// Exchange code for token
	fmt.Println("Exchanging authorization code for tokens...")
	token, err := sdk.OAuthCallback(ctx, provider, result.code, state, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Display token
	logJSONCmd(*cmd, token)
	fmt.Println("\nAuthentication successful! You can now use the access_token for API requests.")

	return nil
}

func startCallbackServer(listener net.Listener, resultChan chan<- oauthCallbackResult) *http.Server {
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	var once sync.Once

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, errorHTML, errParam)
			once.Do(func() {
				resultChan <- oauthCallbackResult{err: fmt.Errorf("oauth error: %s", errParam)}
			})
			return
		}

		if code == "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, errorHTML, "missing authorization code")
			once.Do(func() {
				resultChan <- oauthCallbackResult{err: fmt.Errorf("missing authorization code")}
			})
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML)

		once.Do(func() {
			resultChan <- oauthCallbackResult{code: code, state: state}
		})
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			once.Do(func() {
				resultChan <- oauthCallbackResult{err: fmt.Errorf("server error: %w", err)}
			})
		}
	}()

	return server
}

func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		// Ignore shutdown errors
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
