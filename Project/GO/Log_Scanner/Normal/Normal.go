// Log_Scanner is a Go program that scans log files for specific keywords.
// Config is Externalized to a JSON file for easy modification of keywords.
package main

// bufio: Buffered I/O operations for wrapping around io.Reader and io.Writer
// encoding/json: JSON encoding and decoding as defined in RFC 7159
// fmt: Formatted I/O with functions analogous to C's printf and scanf
// os: Platform-independent interface to operating system functionality
// strings: Functions to manipulate UTF-8 encoded strings
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Config struct to hold the keywords for scanning
type Config struct {
	TargetLogPath string   `json:"target_log_path"`
	Keywords      []string `json:"keywords"`
	LogPattern    string   `json:"log_pattern"`
}

// func main is the entry point of the program
func main() {
	// Load configuration from config.json from the current directory
	configFile, err := loadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	// Scan the log file based on the loaded configuration
	if err := scanLog(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning log: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig reads the configuration from a JSON file and unmarshals it into a Config struct
func loadConfig(path string) (*Config, error) {
	// Open the configuration file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Decode JSON content into Config struct
	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// scanfLog scans the log file for lines containing any of the specified keywords
func scanLog(config *Config) error {
	file, err := os.Open(config.TargetLogPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// [Official Doc: pkg.go.dev/regexp#MustCompile]
	// Compile regex outside the loop to avoid performance penalty.
	// If pattern is empty, we skip regex extraction.
	var regex *regexp.Regexp
	if config.LogPattern != "" {
		regex = regexp.MustCompile(config.LogPattern)
	}

	// Create a buffered scanner to read the file line by line
	const maxCapacity = 10 * 1024 * 1024 // 10 MB
	// Initialize scanner with increased buffer size(64KB) to handle long lines
	buf := make([]byte, 64*1024) // 64 KB initial buffer size
	scanner := bufio.NewScanner(file)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		// [Optimization Logic]
		// 1. Fast Filter: Check keywords first using efficient string search.
		found := false
		for _, keyword := range config.Keywords {
			if strings.Contains(line, keyword) {
				found = true
				break
			}
		}
		if found {
			// 2. Slow Extraction: Apply regex only if keyword is found.
			if regex != nil {
				matches := regex.FindStringSubmatch(line)
				if len(matches) > 1 {
					// Assuming Group 1: Level, Group 2: Code, Group 3: Message
					fmt.Printf("[DETECTED] %v\n", matches[1:])
				} else {
					// If regex does not match, print the raw line
					fmt.Printf("[RAW MATCH] %s\n", line)
				}
			} else {
				// If no regex pattern is provided, just print the line with keyword
				fmt.Printf("[FOUND] %s\n", line)
			}
		}
	}
	// ouput Scan result
	fmt.Println("[SCAN COMPLETE]")
	return scanner.Err()
}
