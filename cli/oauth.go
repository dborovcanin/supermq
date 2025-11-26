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
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	callbackPath    = "/callback"
	localServerPort = "9090"
	callbackTimeout = 5 * time.Minute
	shutdownTimeout = 5 * time.Second
)

const successHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Successful - Magistrala</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: rgb(1, 69, 96);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 48px;
            max-width: 500px;
            width: 100%;
            text-align: center;
            animation: slideIn 0.4s ease-out;
        }
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(-20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        .logo-container {
            margin-bottom: 32px;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .logo {
            width: 300px;
            height: auto;
            animation: logoFadeIn 0.8s ease-out 0.3s both;
        }
        @keyframes logoFadeIn {
            from {
                opacity: 0;
                transform: scale(0.8);
            }
            to {
                opacity: 1;
                transform: scale(1);
            }
        }
        .success-icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 24px;
            background: rgb(1, 69, 96);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            animation: scaleIn 0.5s ease-out 0.5s both;
        }
        @keyframes scaleIn {
            from {
                transform: scale(0);
            }
            to {
                transform: scale(1);
            }
        }
        .checkmark {
            width: 40px;
            height: 40px;
            border: 4px solid white;
            border-radius: 50%;
            border-left-color: transparent;
            border-top-color: transparent;
            transform: rotate(45deg);
        }
        h1 {
            color: rgb(1, 69, 96);
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 16px;
        }
        p {
            color: #4a5568;
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 12px;
        }
        .footer {
            margin-top: 32px;
            padding-top: 24px;
            border-top: 1px solid #e2e8f0;
            color: #a0aec0;
            font-size: 14px;
        }
        @media (max-width: 600px) {
            .container {
                padding: 32px 24px;
            }
            .logo {
                width: 240px;
            }
            h1 {
                font-size: 24px;
            }
            .success-icon,
            .error-icon {
                width: 64px;
                height: 64px;
            }
        }
        @media (max-width: 400px) {
            .container {
                padding: 24px 16px;
            }
            .logo {
                width: 200px;
            }
            h1 {
                font-size: 20px;
            }
            p {
                font-size: 14px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo-container">
            <img src="https://cloud.magistrala.absmach.eu/_next/static/media/Magistrala_logo_landscape_white.59ea595a.svg"
                 alt="Magistrala Logo"
                 class="logo"
                 style="filter: brightness(0) saturate(100%) invert(15%) sepia(84%) saturate(2449%) hue-rotate(173deg) brightness(94%) contrast(101%);">
        </div>

        <div class="success-icon">
            <div class="checkmark"></div>
        </div>

        <h1>Authentication Successful!</h1>
        <p>You have been successfully authenticated.</p>
        <p>You can now close this window and return to the CLI.</p>

        <div class="footer">
            Powered by SuperMQ
        </div>
    </div>
</body>
</html>`

const errorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Failed - Magistrala</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: rgb(1, 69, 96);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 48px;
            max-width: 500px;
            width: 100%;
            text-align: center;
            animation: slideIn 0.4s ease-out;
        }
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(-20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        .logo-container {
            margin-bottom: 32px;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .logo {
            width: 300px;
            height: auto;
            animation: logoFadeIn 0.8s ease-out 0.3s both;
        }
        @keyframes logoFadeIn {
            from {
                opacity: 0;
                transform: scale(0.8);
            }
            to {
                opacity: 1;
                transform: scale(1);
            }
        }
        .error-icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 24px;
            background: #dc2626;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            animation: shake 0.5s ease-out 0.5s both;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-10px); }
            75% { transform: translateX(10px); }
        }
        .cross {
            width: 50px;
            height: 50px;
            position: relative;
        }
        .cross::before,
        .cross::after {
            content: '';
            position: absolute;
            width: 4px;
            height: 50px;
            background: white;
            left: 50%;
            top: 0;
            border-radius: 2px;
        }
        .cross::before {
            transform: translateX(-50%) rotate(45deg);
        }
        .cross::after {
            transform: translateX(-50%) rotate(-45deg);
        }
        h1 {
            color: rgb(1, 69, 96);
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 16px;
        }
        p {
            color: #4a5568;
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 12px;
        }
        .error-message {
            background: #fef2f2;
            border: 1px solid #fecaca;
            border-radius: 8px;
            padding: 16px;
            margin: 24px 0;
            color: #991b1b;
            font-family: monospace;
            font-size: 14px;
            word-break: break-word;
        }
        .footer {
            margin-top: 32px;
            padding-top: 24px;
            border-top: 1px solid #e2e8f0;
            color: #a0aec0;
            font-size: 14px;
        }
        @media (max-width: 600px) {
            .container {
                padding: 32px 24px;
            }
            .logo {
                width: 240px;
            }
            h1 {
                font-size: 24px;
            }
            .success-icon,
            .error-icon {
                width: 64px;
                height: 64px;
            }
        }
        @media (max-width: 400px) {
            .container {
                padding: 24px 16px;
            }
            .logo {
                width: 200px;
            }
            h1 {
                font-size: 20px;
            }
            p {
                font-size: 14px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo-container">
            <img src="https://cloud.magistrala.absmach.eu/_next/static/media/Magistrala_logo_landscape_white.59ea595a.svg"
                 alt="Magistrala Logo"
                 class="logo"
                 style="filter: brightness(0) saturate(100%) invert(15%) sepia(84%) saturate(2449%) hue-rotate(173deg) brightness(94%) contrast(101%);">
        </div>

        <div class="error-icon">
            <div class="cross"></div>
        </div>

        <h1>Authentication Failed</h1>
        <p>We encountered an error during authentication.</p>

        <div class="error-message">
            {{ERROR_MESSAGE}}
        </div>

        <p>Please close this window and try again.</p>

        <div class="footer">
            Powered by SuperMQ
        </div>
    </div>
</body>
</html>`

type oauthCallbackResult struct {
	code  string
	state string
	err   error
}

type browserOpener interface {
	Open(url string) error
}

type defaultBrowserOpener struct{}

func (defaultBrowserOpener) Open(url string) error {
	return openBrowser(url)
}

type callbackServer struct {
	listener net.Listener
	server   *http.Server
}

func newCallbackServer(resultChan chan<- oauthCallbackResult) (*callbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+localServerPort)
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	var once sync.Once
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		handleOAuthCallback(w, r, resultChan, &once)
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			once.Do(func() {
				resultChan <- oauthCallbackResult{err: fmt.Errorf("server error: %w", err)}
			})
		}
	}()

	return &callbackServer{
		listener: listener,
		server:   server,
	}, nil
}

func (cs *callbackServer) Shutdown(cmd *cobra.Command) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := cs.server.Shutdown(ctx); err != nil {
		logErrorCmd(*cmd, err)
	}
}

func performOAuthLogin(cmd *cobra.Command, provider string) error {
	return performOAuthLoginWithBrowser(cmd, provider, defaultBrowserOpener{})
}

func performOAuthLoginWithBrowser(cmd *cobra.Command, provider string, browser browserOpener) error {
	ctx := cmd.Context()
	callbackChan := make(chan oauthCallbackResult, 1)

	server, err := newCallbackServer(callbackChan)
	if err != nil {
		return err
	}
	defer server.Shutdown(cmd)

	callbackURL := fmt.Sprintf("http://127.0.0.1:%s%s", localServerPort, callbackPath)
	authURL, state, err := sdk.OAuthAuthorizationURL(ctx, provider, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to get authorization URL: %w", err)
	}

	printAuthInstructions(authURL)
	if err := browser.Open(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
	}

	fmt.Println("Waiting for authentication callback...")

	result, err := waitForCallback(callbackChan)
	if err != nil {
		return err
	}

	fmt.Println("Exchanging authorization code for tokens...")
	token, err := sdk.OAuthCallback(ctx, provider, result.code, state, callbackURL)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	logJSONCmd(*cmd, token)
	fmt.Println("\nAuthentication successful! You can now use the access_token for API requests.")

	return nil
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, resultChan chan<- oauthCallbackResult, once *sync.Once) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	w.Header().Set("Content-Type", "text/html")

	if errParam != "" {
		html := strings.Replace(errorHTML, "{{ERROR_MESSAGE}}", errParam, 1)
		fmt.Fprint(w, html)
		once.Do(func() {
			resultChan <- oauthCallbackResult{err: fmt.Errorf("oauth error: %s", errParam)}
		})
		return
	}

	if code == "" {
		html := strings.Replace(errorHTML, "{{ERROR_MESSAGE}}", "missing authorization code", 1)
		fmt.Fprint(w, html)
		once.Do(func() {
			resultChan <- oauthCallbackResult{err: fmt.Errorf("missing authorization code")}
		})
		return
	}

	fmt.Fprint(w, successHTML)
	once.Do(func() {
		resultChan <- oauthCallbackResult{code: code, state: state}
	})
}

func printAuthInstructions(authURL string) {
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, please visit:\n%s\n\n", authURL)
}

func waitForCallback(callbackChan <-chan oauthCallbackResult) (oauthCallbackResult, error) {
	select {
	case result := <-callbackChan:
		if result.err != nil {
			return oauthCallbackResult{}, fmt.Errorf("callback error: %w", result.err)
		}
		return result, nil
	case <-time.After(callbackTimeout):
		return oauthCallbackResult{}, fmt.Errorf("authentication timeout after %v", callbackTimeout)
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
