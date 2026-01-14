package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

// extractChromeCookies extracts LinkedIn cookies from Chrome.
func extractChromeCookies() ([]Cookie, error) {
	cookiePath, err := findChromeCookiesPath()
	if err != nil {
		return nil, err
	}

	// Chrome locks the database, so copy it to a temp file.
	tmpFile, err := copyToTemp(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy cookies database: %w", err)
	}
	defer os.Remove(tmpFile)

	// Get decryption key.
	key, err := getChromeDecryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get Chrome decryption key: %w", err)
	}

	return readChromeCookies(tmpFile, key)
}

// findChromeCookiesPath locates the Chrome cookies database.
func findChromeCookiesPath() (string, error) {
	var basePath string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		basePath = filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Try Chrome first, then Chromium.
		chromePath := filepath.Join(home, ".config", "google-chrome")
		if _, err := os.Stat(chromePath); err == nil {
			basePath = chromePath
		} else {
			chromiumPath := filepath.Join(home, ".config", "chromium")
			if _, err := os.Stat(chromiumPath); err == nil {
				basePath = chromiumPath
			} else {
				return "", fmt.Errorf("Chrome/Chromium config not found. Is Chrome installed?")
			}
		}
	default:
		return "", fmt.Errorf("Chrome cookie extraction not supported on %s", runtime.GOOS)
	}

	// Check Default profile first.
	cookiePath := filepath.Join(basePath, "Default", "Cookies")
	if _, err := os.Stat(cookiePath); err == nil {
		return cookiePath, nil
	}

	// Try Network/Cookies (newer Chrome versions).
	networkCookiePath := filepath.Join(basePath, "Default", "Network", "Cookies")
	if _, err := os.Stat(networkCookiePath); err == nil {
		return networkCookiePath, nil
	}

	return "", fmt.Errorf("Chrome cookies database not found")
}

// getChromeDecryptionKey retrieves the key used to decrypt Chrome cookies.
func getChromeDecryptionKey() ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return getChromeKeyMacOS()
	case "linux":
		return getChromeKeyLinux()
	default:
		return nil, fmt.Errorf("Chrome decryption not supported on %s", runtime.GOOS)
	}
}

// getChromeKeyMacOS retrieves the Chrome encryption key from macOS Keychain.
func getChromeKeyMacOS() ([]byte, error) {
	// Use security command to get the key from Keychain.
	cmd := exec.Command("security", "find-generic-password",
		"-w", // Print password only
		"-s", "Chrome Safe Storage",
		"-a", "Chrome",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Chrome key from Keychain: %w. Make sure Chrome has stored its key in Keychain", err)
	}

	password := strings.TrimSpace(string(output))

	// Derive key using PBKDF2.
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1003, 16, sha1.New)

	return key, nil
}

// getChromeKeyLinux retrieves the Chrome encryption key on Linux.
func getChromeKeyLinux() ([]byte, error) {
	// On Linux, Chrome uses either:
	// 1. GNOME Keyring (if available)
	// 2. A hardcoded key "peanuts"

	// Try to get from GNOME Keyring first using secret-tool.
	cmd := exec.Command("secret-tool", "lookup",
		"application", "chrome",
	)

	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		password := strings.TrimSpace(string(output))
		salt := []byte("saltysalt")
		key := pbkdf2.Key([]byte(password), salt, 1, 16, sha1.New)
		return key, nil
	}

	// Fallback to hardcoded key.
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte("peanuts"), salt, 1, 16, sha1.New)

	return key, nil
}

// readChromeCookies reads and decrypts cookies from a Chrome cookies database.
func readChromeCookies(dbPath string, key []byte) ([]Cookie, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open cookies database: %w", err)
	}
	defer db.Close()

	// Query LinkedIn cookies.
	query := `
		SELECT name, encrypted_value, host_key, path, expires_utc, is_secure, is_httponly
		FROM cookies
		WHERE host_key LIKE '%linkedin.com'
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var name, host, path string
		var encryptedValue []byte
		var expiresUTC int64
		var isSecure, isHTTPOnly int

		if err := rows.Scan(&name, &encryptedValue, &host, &path, &expiresUTC, &isSecure, &isHTTPOnly); err != nil {
			continue
		}

		// Decrypt cookie value.
		value, err := decryptChromeCookie(encryptedValue, key)
		if err != nil {
			// Try unencrypted value.
			value = string(encryptedValue)
		}

		// Chrome stores time as microseconds since 1601-01-01.
		// Convert to Unix timestamp.
		expiresAt := chromeTimeToUnix(expiresUTC)

		cookies = append(cookies, Cookie{
			Domain:     host,
			Name:       name,
			Value:      value,
			Path:       path,
			ExpiresAt:  expiresAt,
			IsSecure:   isSecure == 1,
			IsHTTPOnly: isHTTPOnly == 1,
		})
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no LinkedIn cookies found in Chrome. Make sure you're logged into LinkedIn")
	}

	return cookies, nil
}

// decryptChromeCookie decrypts a Chrome cookie value.
func decryptChromeCookie(encrypted []byte, key []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}

	// Check for encryption version prefix.
	if runtime.GOOS == "darwin" && len(encrypted) > 3 && string(encrypted[:3]) == "v10" {
		// v10 encryption (AES-128-CBC).
		return decryptV10Cookie(encrypted[3:], key)
	}

	if runtime.GOOS == "linux" && len(encrypted) > 3 && string(encrypted[:3]) == "v11" {
		// v11 encryption (AES-128-CBC).
		return decryptV10Cookie(encrypted[3:], key)
	}

	if runtime.GOOS == "linux" && len(encrypted) > 3 && string(encrypted[:3]) == "v10" {
		return decryptV10Cookie(encrypted[3:], key)
	}

	// Unencrypted or unknown format.
	return string(encrypted), nil
}

// decryptV10Cookie decrypts a v10 encrypted cookie using AES-128-CBC.
func decryptV10Cookie(encrypted []byte, key []byte) (string, error) {
	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	// IV is all spaces for Chrome.
	iv := []byte("                ") // 16 spaces

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// Ensure encrypted data is multiple of block size.
	if len(encrypted)%aes.BlockSize != 0 {
		return "", fmt.Errorf("encrypted data not aligned")
	}

	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// Remove PKCS7 padding.
	decrypted = removePKCS7Padding(decrypted)

	return string(decrypted), nil
}

// removePKCS7Padding removes PKCS7 padding from decrypted data.
func removePKCS7Padding(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > aes.BlockSize {
		return data
	}
	// Verify padding bytes.
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return data
		}
	}
	return data[:len(data)-padding]
}

// chromeTimeToUnix converts Chrome's timestamp format to Unix time.
// Chrome uses microseconds since 1601-01-01.
func chromeTimeToUnix(chromeTime int64) time.Time {
	if chromeTime == 0 {
		return time.Time{}
	}
	// Microseconds between 1601-01-01 and 1970-01-01.
	const epochDiff = 11644473600000000
	unixMicro := chromeTime - epochDiff
	return time.Unix(unixMicro/1000000, (unixMicro%1000000)*1000)
}

// decodeBase64 is a helper for base64 decoding.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
