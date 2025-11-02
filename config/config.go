package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration values
type Config struct {
	ServerHostname string
	ServerPort     uint16
	ServerAddress  string
	ServerDomain   string
	ClientHostname string
	ClientPort     uint16
	Relay          bool
	RequireTLS     bool
	// TLS configuration for STARTTLS
	TLSEnabled bool   // Enable STARTTLS (advertises it in EHLO)
	TLSCertFile string // Path to TLS certificate file (e.g., "cert.pem")
	TLSKeyFile  string // Path to TLS private key file (e.g., "key.pem")
}

var globalConfig *Config

// LoadConfig loads configuration from environment variables or .env file
func LoadConfig() (*Config, error) {
	// Try to load .env file first
	loadEnvFile(".env")

	config := &Config{
		ServerHostname: getEnv("SMTP_SERVER_HOSTNAME", "localhost"),
		ServerPort:     uint16(getEnvAsInt("SMTP_SERVER_PORT", 2525)),
		ServerAddress:  getEnv("SMTP_SERVER_ADDRESS", "0.0.0.0"),
		ServerDomain:   getEnv("SMTP_SERVER_DOMAIN", "localhost"),
		ClientHostname: getEnv("SMTP_CLIENT_HOSTNAME", "localhost"),
		ClientPort:     uint16(getEnvAsInt("SMTP_CLIENT_PORT", 587)),
		Relay:          getEnvAsBool("SMTP_RELAY", false),
		RequireTLS:     getEnvAsBool("SMTP_REQUIRE_TLS", false),
		// TLS configuration
		TLSEnabled:  getEnvAsBool("SMTP_TLS_ENABLED", false),
		TLSCertFile: getEnv("SMTP_TLS_CERT_FILE", "cert.pem"),
		TLSKeyFile:  getEnv("SMTP_TLS_KEY_FILE", "key.pem"),
	}

	globalConfig = config
	return config, nil
}

// GetConfig returns the global configuration
func GetConfig() *Config {
	if globalConfig == nil {
		LoadConfig()
	}
	return globalConfig
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// getEnvAsBool gets an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		// Also accept "1", "true", "yes" as true
		valueLower := strings.ToLower(value)
		if valueLower == "1" || valueLower == "true" || valueLower == "yes" {
			return true
		}
		return false
	}
	return boolValue
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		// .env file doesn't exist, that's okay - use defaults
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Remove quotes if present
			if len(value) > 0 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}
			
			// Only set if not already set in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}

	return scanner.Err()
}

// PrintConfig prints the current configuration (without sensitive data)
func (c *Config) PrintConfig() {
	fmt.Printf("SMTP Server Configuration:\n")
	fmt.Printf("  Hostname: %s\n", c.ServerHostname)
	fmt.Printf("  Port: %d\n", c.ServerPort)
	fmt.Printf("  Address: %s\n", c.ServerAddress)
	fmt.Printf("  Domain: %s\n", c.ServerDomain)
	fmt.Printf("  Relay: %v\n", c.Relay)
	fmt.Printf("  Require TLS: %v\n", c.RequireTLS)
	fmt.Printf("  TLS Enabled (STARTTLS): %v\n", c.TLSEnabled)
	if c.TLSEnabled {
		fmt.Printf("  TLS Cert File: %s\n", c.TLSCertFile)
		fmt.Printf("  TLS Key File: %s\n", c.TLSKeyFile)
	}
	fmt.Printf("  Client Hostname: %s\n", c.ClientHostname)
	fmt.Printf("  Client Port: %d\n", c.ClientPort)
}

