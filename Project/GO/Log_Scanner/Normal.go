/*
Normal.go

	A Go program to scan large log files for specific keywords in a sequential manner.
	It reads the log file line by line and checks for the presence of user-defined keywords.
	The program reads a configuration file (config.json) to get the target log file path and keywords to search for.
*/
package main

/*
bufio: Buffered I/O operations for wrapping around io.Reader and io.Writer
encoding/json: JSON encoding and decoding as defined in RFC 7159
fmt: Formatted I/O with functions analogous to C's printf and scanf
os: Platform-independent interface to operating system functionality
strings: Functions to manipulate UTF-8 encoded strings
*/
import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Function to run the log scanner in normal (sequential) mode
func runNormalMode(config *Config) error {
	file, err := os.Open(config.TargetLogPath)
	// Handle file open error
	if err != nil {
		fmt.Printf("[ERROR] Failed to open file: %v\n", err)
		return err
	}
	defer file.Close()
	// Compile regex pattern if provided
	var regex *regexp.Regexp
	// Handle regex compilation error
	if config.LogPattern != "" {
		regex = regexp.MustCompile(config.LogPattern)
	}
	// Buffered scanner for efficient line-by-line reading of the log file
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	fmt.Println("[SYSTEM] Starting Sequential Scan...")
	// Scan through each line of the log file and check for keywords and patterns
	for scanner.Scan() {
		line := scanner.Text()
		found := false
		// Check for presence of any keyword in the current line
		for _, keyword := range config.Keywords {
			if strings.Contains(line, keyword) {
				found = true
				break
			}
		}
		// If a keyword is found, check for regex pattern and output results accordingly
		if found {
			if regex != nil {
				matches := regex.FindStringSubmatch(line)
				if len(matches) > 1 {
					fmt.Printf("[DETECTED] %v\n", matches[1:])
				} else {
					fmt.Printf("[RAW] %s\n", line)
				}
			} else {
				fmt.Printf("[FOUND] %s\n", line)
			}
		}
	}

	fmt.Println("[SYSTEM] Scan Complete.")
	return scanner.Err()
}
