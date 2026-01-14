// Package commands provides CLI command implementations.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pp/lnk/internal/api"
	"github.com/pp/lnk/internal/auth"
	"github.com/spf13/cobra"
)

var (
	authBrowser  string
	authEmail    string
	authPassword string
)

// NewAuthCmd creates the auth command group.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  `Commands for authenticating with LinkedIn.`,
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthLogoutCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with LinkedIn",
		Long: `Authenticate with LinkedIn using browser cookies.

Auto-detect default browser (recommended):
  lnk auth login

Specify browser manually:
  lnk auth login --browser safari
  lnk auth login --browser chrome
  lnk auth login --browser helium
  lnk auth login --browser brave
  lnk auth login --browser arc

Environment variables:
  Set LNK_LI_AT and LNK_JSESSIONID, then run:
  lnk auth login --env

Supported browsers: safari, chrome, chromium, firefox, brave, edge, arc, helium, opera, vivaldi

Note: Browser cookie extraction may require granting Full Disk Access
to your terminal application in System Preferences > Privacy & Security.`,
		RunE: runAuthLogin,
	}

	cmd.Flags().StringVarP(&authBrowser, "browser", "b", "", "Browser to extract cookies from (auto-detected if not specified)")
	cmd.Flags().StringVarP(&authEmail, "email", "e", "", "LinkedIn email (for password auth)")
	cmd.Flags().StringVarP(&authPassword, "password", "p", "", "LinkedIn password (for password auth)")
	cmd.Flags().Bool("env", false, "Use environment variables for authentication")

	return cmd
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	useEnv, _ := cmd.Flags().GetBool("env")

	var creds *api.Credentials
	var err error
	var browserUsed auth.Browser

	switch {
	case useEnv:
		creds, err = auth.FromEnvironment()
	case authBrowser != "":
		browserUsed = auth.Browser(strings.ToLower(authBrowser))
		creds, err = auth.ExtractLinkedInCookies(browserUsed)
	case authEmail != "" && authPassword != "":
		err = fmt.Errorf("username/password authentication not yet implemented")
	default:
		// Auto-detect default browser.
		browserUsed, err = auth.DetectDefaultBrowser()
		if err != nil {
			return outputError(jsonOutput, "BROWSER_DETECT_FAILED",
				fmt.Sprintf("could not detect default browser: %v. Use --browser to specify manually", err))
		}
		if !jsonOutput {
			fmt.Printf("Detected browser: %s\n", browserUsed)
		}
		creds, err = auth.ExtractLinkedInCookies(browserUsed)
	}

	if err != nil {
		return outputError(jsonOutput, "LOGIN_FAILED", err.Error())
	}

	// Validate credentials by checking if they look valid.
	if !creds.IsValid() {
		return outputError(jsonOutput, "INVALID_CREDENTIALS", "extracted credentials are invalid or expired")
	}

	// Store credentials.
	store, err := auth.NewStore()
	if err != nil {
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	if err := store.Save(creds); err != nil {
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	if jsonOutput {
		return outputJSON(api.Response[map[string]any]{
			Success: true,
			Data: map[string]any{
				"message":    "Successfully authenticated",
				"storedAt":   store.Path(),
				"hasLiAt":    creds.LiAt != "",
				"hasJSessID": creds.JSessID != "",
			},
		})
	}

	fmt.Println("Successfully authenticated with LinkedIn!")
	fmt.Printf("Credentials stored at: %s\n", store.Path())
	return nil
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  `Check if you are currently authenticated with LinkedIn.`,
		RunE:  runAuthStatus,
	}
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	store, err := auth.NewStore()
	if err != nil {
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	creds, err := store.Load()
	if err != nil {
		if err == auth.ErrNoCredentials {
			if jsonOutput {
				return outputJSON(api.Response[map[string]any]{
					Success: true,
					Data: map[string]any{
						"authenticated": false,
						"message":       "Not authenticated. Run: lnk auth login",
					},
				})
			}
			fmt.Println("Not authenticated.")
			fmt.Println("Run: lnk auth login --browser safari")
			return nil
		}
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	isValid := creds.IsValid()

	if jsonOutput {
		data := map[string]any{
			"authenticated": true,
			"valid":         isValid,
			"hasLiAt":       creds.LiAt != "",
			"hasJSessID":    creds.JSessID != "",
			"storedAt":      store.Path(),
		}
		if !creds.ExpiresAt.IsZero() {
			data["expiresAt"] = creds.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		}
		return outputJSON(api.Response[map[string]any]{
			Success: true,
			Data:    data,
		})
	}

	if isValid {
		fmt.Println("Authenticated with LinkedIn.")
		fmt.Printf("Credentials stored at: %s\n", store.Path())
		if !creds.ExpiresAt.IsZero() {
			fmt.Printf("Expires: %s\n", creds.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("Credentials are expired or invalid.")
		fmt.Println("Run: lnk auth login --browser safari")
	}

	return nil
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored credentials",
		Long:  `Remove stored LinkedIn credentials.`,
		RunE:  runAuthLogout,
	}
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	store, err := auth.NewStore()
	if err != nil {
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	if err := store.Delete(); err != nil {
		return outputError(jsonOutput, "STORE_ERROR", err.Error())
	}

	if jsonOutput {
		return outputJSON(api.Response[map[string]any]{
			Success: true,
			Data: map[string]any{
				"message": "Successfully logged out",
			},
		})
	}

	fmt.Println("Successfully logged out.")
	return nil
}

// Helper functions for output formatting.

func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func outputError(jsonOutput bool, code, message string) error {
	if jsonOutput {
		outputJSON(api.Response[any]{
			Success: false,
			Error: &api.Error{
				Code:    code,
				Message: message,
			},
		})
		os.Exit(1)
	}
	return fmt.Errorf("%s", message)
}
