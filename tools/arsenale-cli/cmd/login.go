package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var loginServer string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via browser-based device authorization",
	Long:  `Initiates the OAuth 2.0 Device Authorization Grant flow. Opens a browser for authentication and polls for the token.`,
	Run:   runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&loginServer, "server", "s", "", "Server URL to authenticate against")
}

func runLogin(cmd *cobra.Command, args []string) {
	cfg := getCfg()

	if loginServer != "" {
		cfg.ServerURL = loginServer
	}
	if url := os.Getenv("ARSENALE_SERVER"); url != "" && loginServer == "" {
		cfg.ServerURL = url
	}

	fmt.Printf("Authenticating with %s ...\n\n", cfg.ServerURL)

	// Step 1: Initiate device authorization
	respBody, status, err := doRequest("POST", "/api/cli/auth/device", nil, cfg)
	if err != nil {
		fatal("failed to initiate device authorization: %v", err)
	}
	if status != 200 {
		fatal("server returned HTTP %d: %s", status, string(respBody))
	}

	var deviceResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(respBody, &deviceResp); err != nil {
		fatal("failed to parse device auth response: %v", err)
	}

	fmt.Println("To authenticate, open the following URL in your browser:")
	fmt.Printf("\n  %s\n\n", deviceResp.VerificationURIComplete)
	fmt.Printf("Or go to %s and enter code: %s\n\n", deviceResp.VerificationURI, deviceResp.UserCode)

	if err := openBrowser(deviceResp.VerificationURIComplete); err != nil {
		fmt.Println("(Could not open browser automatically, please copy the URL above)")
	} else {
		fmt.Println("Browser opened. Waiting for authorization...")
	}

	// Step 2: Poll for token
	interval := time.Duration(deviceResp.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		body := map[string]string{
			"device_code": deviceResp.DeviceCode,
		}

		respBody, status, err = doRequest("POST", "/api/cli/auth/device/token", body, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error polling for token: %v\n", err)
			continue
		}

		if status == 200 {
			var tokenResp struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				TokenType    string `json:"token_type"`
				User         struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"user"`
			}
			if err := json.Unmarshal(respBody, &tokenResp); err != nil {
				fatal("failed to parse token response: %v", err)
			}

			cfg.AccessToken = tokenResp.AccessToken
			cfg.RefreshToken = tokenResp.RefreshToken
			cfg.TokenExpiry = time.Now().Add(14 * time.Minute).Format(time.RFC3339)
			// Refresh cached tenant context after login so old local state
			// does not survive a stack reset or tenant switch.
			cfg.TenantID = ""
			_ = cfg.resolveTenantID()

			if err := saveConfig(cfg); err != nil {
				fatal("failed to save config: %v", err)
			}

			fmt.Printf("\nAuthenticated as %s\n", tokenResp.User.Email)
			fmt.Printf("Credentials saved to %s\n", configPath())
			return
		}

		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			switch errResp.Error {
			case "authorization_pending":
				fmt.Print(".")
				continue
			case "slow_down":
				interval += 5 * time.Second
				continue
			case "expired_token":
				fatal("device code expired. Please try again.")
			default:
				fatal("%s", errResp.Error)
			}
		}
	}

	fatal("device authorization timed out. Please try again.")
}

func openBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	return c.Start()
}
