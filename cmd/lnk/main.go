// Package main is the entry point for the lnk CLI.
//
// lnk is a fast LinkedIn CLI for posting, reading, and messaging
// via LinkedIn's Voyager API. It is designed to work seamlessly
// with AI agents through structured JSON output.
//
// Usage:
//
//	lnk [command] [flags]
//
// Example:
//
//	lnk auth login --browser safari
//	lnk profile me --json
//	lnk post create "Hello LinkedIn!"
package main

func main() {
	Execute()
}
