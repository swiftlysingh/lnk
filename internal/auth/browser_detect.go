package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectDefaultBrowser attempts to detect the user's default browser.
func DetectDefaultBrowser() (Browser, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectDefaultBrowserMacOS()
	case "linux":
		return detectDefaultBrowserLinux()
	default:
		return "", fmt.Errorf("browser detection not supported on %s", runtime.GOOS)
	}
}

// detectDefaultBrowserMacOS detects the default browser on macOS.
func detectDefaultBrowserMacOS() (Browser, error) {
	// Use Launch Services to get the default HTTP handler.
	cmd := exec.Command("defaults", "read",
		"com.apple.LaunchServices/com.apple.launchservices.secure",
		"LSHandlers")

	output, err := cmd.Output()
	if err == nil {
		outputStr := string(output)

		// Look for HTTP handler bundle ID.
		if strings.Contains(outputStr, "com.helium-browser") ||
			strings.Contains(outputStr, "browser.helium") ||
			strings.Contains(outputStr, "helium") {
			return BrowserHelium, nil
		}
		if strings.Contains(outputStr, "com.google.chrome") {
			return BrowserChrome, nil
		}
		if strings.Contains(outputStr, "org.mozilla.firefox") {
			return BrowserFirefox, nil
		}
		if strings.Contains(outputStr, "com.apple.safari") {
			return BrowserSafari, nil
		}
		if strings.Contains(outputStr, "com.brave.browser") ||
			strings.Contains(outputStr, "com.brave.Browser") {
			return BrowserBrave, nil
		}
		if strings.Contains(outputStr, "com.microsoft.edgemac") {
			return BrowserEdge, nil
		}
		if strings.Contains(outputStr, "com.operasoftware.Opera") {
			return BrowserOpera, nil
		}
		if strings.Contains(outputStr, "com.vivaldi.Vivaldi") {
			return BrowserVivaldi, nil
		}
		if strings.Contains(outputStr, "org.chromium.Chromium") {
			return BrowserChromium, nil
		}
		if strings.Contains(outputStr, "com.arc.browser") ||
			strings.Contains(outputStr, "company.thebrowser.Browser") {
			return BrowserArc, nil
		}
	}

	// Fallback: check which browsers are installed and pick one.
	return findInstalledBrowserMacOS()
}

// findInstalledBrowserMacOS finds an installed browser on macOS.
func findInstalledBrowserMacOS() (Browser, error) {
	home, _ := os.UserHomeDir()

	// Check for browsers in order of preference.
	browsers := []struct {
		browser Browser
		paths   []string
	}{
		{BrowserHelium, []string{
			filepath.Join(home, "Library", "Application Support", "Helium"),
			"/Applications/Helium.app",
		}},
		{BrowserChrome, []string{
			filepath.Join(home, "Library", "Application Support", "Google", "Chrome"),
			"/Applications/Google Chrome.app",
		}},
		{BrowserFirefox, []string{
			filepath.Join(home, "Library", "Application Support", "Firefox"),
			"/Applications/Firefox.app",
		}},
		{BrowserBrave, []string{
			filepath.Join(home, "Library", "Application Support", "BraveSoftware", "Brave-Browser"),
			"/Applications/Brave Browser.app",
		}},
		{BrowserArc, []string{
			filepath.Join(home, "Library", "Application Support", "Arc"),
			"/Applications/Arc.app",
		}},
		{BrowserEdge, []string{
			filepath.Join(home, "Library", "Application Support", "Microsoft Edge"),
			"/Applications/Microsoft Edge.app",
		}},
		{BrowserSafari, []string{
			"/Applications/Safari.app",
		}},
	}

	for _, b := range browsers {
		for _, p := range b.paths {
			if _, err := os.Stat(p); err == nil {
				return b.browser, nil
			}
		}
	}

	return "", fmt.Errorf("no supported browser found")
}

// detectDefaultBrowserLinux detects the default browser on Linux.
func detectDefaultBrowserLinux() (Browser, error) {
	// Try xdg-settings.
	cmd := exec.Command("xdg-settings", "get", "default-web-browser")
	output, err := cmd.Output()
	if err == nil {
		desktop := strings.ToLower(strings.TrimSpace(string(output)))

		if strings.Contains(desktop, "chrome") || strings.Contains(desktop, "google-chrome") {
			return BrowserChrome, nil
		}
		if strings.Contains(desktop, "chromium") {
			return BrowserChromium, nil
		}
		if strings.Contains(desktop, "firefox") {
			return BrowserFirefox, nil
		}
		if strings.Contains(desktop, "brave") {
			return BrowserBrave, nil
		}
		if strings.Contains(desktop, "edge") {
			return BrowserEdge, nil
		}
		if strings.Contains(desktop, "opera") {
			return BrowserOpera, nil
		}
		if strings.Contains(desktop, "vivaldi") {
			return BrowserVivaldi, nil
		}
	}

	// Fallback: check which browsers are installed.
	return findInstalledBrowserLinux()
}

// findInstalledBrowserLinux finds an installed browser on Linux.
func findInstalledBrowserLinux() (Browser, error) {
	home, _ := os.UserHomeDir()

	browsers := []struct {
		browser Browser
		paths   []string
	}{
		{BrowserChrome, []string{
			filepath.Join(home, ".config", "google-chrome"),
		}},
		{BrowserChromium, []string{
			filepath.Join(home, ".config", "chromium"),
		}},
		{BrowserFirefox, []string{
			filepath.Join(home, ".mozilla", "firefox"),
		}},
		{BrowserBrave, []string{
			filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser"),
		}},
		{BrowserEdge, []string{
			filepath.Join(home, ".config", "microsoft-edge"),
		}},
	}

	for _, b := range browsers {
		for _, p := range b.paths {
			if _, err := os.Stat(p); err == nil {
				return b.browser, nil
			}
		}
	}

	return "", fmt.Errorf("no supported browser found")
}
