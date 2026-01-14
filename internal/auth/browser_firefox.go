package auth

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// extractFirefoxCookies extracts LinkedIn cookies from Firefox.
func extractFirefoxCookies() ([]Cookie, error) {
	profilePath, err := findFirefoxProfile()
	if err != nil {
		return nil, err
	}

	cookiePath := filepath.Join(profilePath, "cookies.sqlite")

	// Firefox may lock the database, so copy it to a temp file.
	tmpFile, err := copyToTemp(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy cookies database: %w", err)
	}
	defer os.Remove(tmpFile)

	return readFirefoxCookies(tmpFile)
}

// findFirefoxProfile locates the default Firefox profile directory.
func findFirefoxProfile() (string, error) {
	var profilesDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		profilesDir = filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		profilesDir = filepath.Join(home, ".mozilla", "firefox")
	default:
		return "", fmt.Errorf("Firefox cookie extraction not supported on %s", runtime.GOOS)
	}

	// Find the default profile (ends with .default or .default-release).
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("Firefox profiles directory not found. Is Firefox installed?")
		}
		return "", fmt.Errorf("failed to read Firefox profiles: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Look for default profile.
		if filepath.Ext(name) == ".default" || filepath.Ext(name) == ".default-release" ||
			len(name) > 8 && (name[len(name)-8:] == ".default" || name[len(name)-16:] == ".default-release") {
			return filepath.Join(profilesDir, name), nil
		}
	}

	// Fallback: use the first profile found.
	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join(profilesDir, entry.Name())
			// Check if cookies.sqlite exists.
			if _, err := os.Stat(filepath.Join(path, "cookies.sqlite")); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no Firefox profile found")
}

// readFirefoxCookies reads cookies from a Firefox cookies.sqlite file.
func readFirefoxCookies(dbPath string) ([]Cookie, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open cookies database: %w", err)
	}
	defer db.Close()

	// Query LinkedIn cookies.
	query := `
		SELECT name, value, host, path, expiry, isSecure, isHttpOnly
		FROM moz_cookies
		WHERE host LIKE '%linkedin.com'
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var name, value, host, path string
		var expiry int64
		var isSecure, isHTTPOnly int

		if err := rows.Scan(&name, &value, &host, &path, &expiry, &isSecure, &isHTTPOnly); err != nil {
			continue
		}

		cookies = append(cookies, Cookie{
			Domain:     host,
			Name:       name,
			Value:      value,
			Path:       path,
			ExpiresAt:  time.Unix(expiry, 0),
			IsSecure:   isSecure == 1,
			IsHTTPOnly: isHTTPOnly == 1,
		})
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no LinkedIn cookies found in Firefox. Make sure you're logged into LinkedIn")
	}

	return cookies, nil
}

// copyToTemp copies a file to a temporary location.
func copyToTemp(src string) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cookies file not found at %s", src)
		}
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied reading cookies file. Close Firefox and try again")
		}
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "lnk-cookies-*.sqlite")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}
