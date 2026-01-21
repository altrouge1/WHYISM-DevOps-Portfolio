/*
   Abnormal.go
   A Go program to scan large log files for abnormal patterns based on user-defined keywords and regex patterns.
   It processes the log file in parallel using goroutines for efficiency.
   The program reads a configuration file (config.json) to get the target log file path,
   keywords to search for, and an optional regex pattern to extract additional information from matched log lines.
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

/*
	bufio: Provides buffered I/O operations for efficient reading of files.
	encoding/json: Used to parse JSON configuration files.
	fmt: For formatted I/O operations like printing to console.
	os: Provides functions to interact with the operating system, such as file handling.
	regexp: For compiling and using regular expressions.
	runtime: To get information about the Go runtime, such as number of CPU cores.
	strings: For string manipulation functions.
	sync: Provides synchronization primitives like WaitGroup for managing goroutines.
*/

// Config holds the configuration for the log scanner
// including target log path, keywords to search for, and optional regex pattern to extract additional information from matched log lines.
type Config struct {
	TargetLogPath string   `json:"target_log_path"`
	Keywords      []string `json:"keywords"`
	LogPattern    string   `json:"log_pattern"`
}

// Main function to execute the log scanner program
func main() {
	// Load configuration file
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err) // Print error to stderr
		os.Exit(1)
	}
	// log file Path Finder
	file, err := os.Open(config.TargetLogPath) // Open the log file for reading
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err) // Print error to stderr
		os.Exit(1)
	}
	defer file.Close() // Ensure the file is closed when done

	// Get file information
	stat, err := file.Stat() // Get file statistics
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stat Error: %v\n", err) // Print error to stderr
		os.Exit(1)
	}

	// Read Abnormaly size log file.
	fileSize := stat.Size()                   // Get the size of the log file
	numworkers := runtime.NumCPU()            // Determine number of CPU cores for parallel processing
	chunkSize := fileSize / int64(numworkers) // Calculate chunk size for each worker

	// Initialize WaitGroup
	var waitgroup sync.WaitGroup

	// Print system information & Log file size & Workers
	fmt.Printf("[SYSTEM] Workers: %d | File Size: %d GB\n", numworkers, fileSize/(1024*1024*1024))
	// Print number of workers and file size in GB Size

	// Compile regex pattern if provided in config file
	var regexpCompiled *regexp.Regexp // Initialize regex variable
	if config.LogPattern != "" {      // Check if regex pattern is provided
		regexpCompiled = regexp.MustCompile(config.LogPattern) // Compile the regex pattern
		fmt.Println("[SYSTEM] Regex pattern compiled successfully.")
	}

	// Start workers to process log file chunks concurrently
	// Iterate through each worker and assign log file chunks to process concurrently
	for worker := 0; worker < numworkers; worker++ { // Iterate through each worker
		start := int64(worker) * chunkSize // Calculate start position for each chunk
		size := chunkSize                  // Default chunk size
		// Adjust size for the last worker to cover remaining bytes
		if worker == numworkers-1 { // Last worker
			size = fileSize - start // Adjust size to cover remaining bytes
		}
		waitgroup.Add(1)                     // Increment WaitGroup counter
		go func(id int, start, size int64) { // Start goroutine for each worker
			defer waitgroup.Done()                                                               // Decrement WaitGroup counter when done
			processChunk(config.TargetLogPath, start, size, config.Keywords, regexpCompiled, id) // Process the log chunk
		}(worker, start, size) // Pass worker ID, start, and size to the goroutine
	}

	// Wait for all workers to finish
	waitgroup.Wait() // Wait for all goroutines to complete
	fmt.Println("[SYSTEM] Parallel Log scanning completed.")
}

// Load configuration from JSON file
// Returns a Config struct and error if any occurs during loading or parsing
func loadConfig(path string) (*Config, error) { // Load configuration from JSON file
	// Open configuration file
	file, err := os.Open(path) // Open the config file
	if err != nil {            // Handle file open error
		return nil, err // Return error if file cannot be opened
	}
	defer file.Close() // Ensure the file is closed when done

	// Decode JSON configuration
	var configFile Config                                             // Initialize Config struct variable
	if err := json.NewDecoder(file).Decode(&configFile); err != nil { // Decode JSON into Config struct
		return nil, err // Return error if decoding fails
	}
	return &configFile, nil // Return the loaded Config struct
}

// Process a chunk of the log file concurrently with goroutines
func processChunk(path string, start, size int64, keywords []string, regexpCompiled *regexp.Regexp, workerID int) { // Process a chunk of the log file
	file, err := os.Open(path) // Open the log file for reading

	// Handle file open error
	if err != nil {
		fmt.Fprintf(os.Stderr, "Worker %d: Error opening log file: %v\n", workerID, err)
		return
	}
	defer file.Close() // Ensure the file is closed when done

	// Seek to the start of the chunk & handle seek error
	// Move the file pointer to the start of the chunk to be processed by this worker goroutine
	if _, err := file.Seek(start, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Worker %d: Error seeking log file: %v\n", workerID, err)
		return
	}

	// Create a buffered reader for efficient reading of the file chunk
	reader := bufio.NewReader(file)

	// If not at the beginning of the file, discard the first line to avoid partial line processing
	if start > 0 {
		_, err := reader.ReadString('\n') // Discard partial line
		// Handle read error
		if err != nil {
			fmt.Fprintf(os.Stderr, "Worker %d: Error reading log file: %v\n", workerID, err)
			return
		}
	}

	// Create a scanner to read the file line by line within the chunk
	scanner := bufio.NewScanner(reader) // Initialize scanner for the reader
	// Set a larger buffer size for the scanner to handle long log lines
	buf := make([]byte, 64*1024)   // 64KB buffer
	scanner.Buffer(buf, 1024*1024) // 1MB max token size

	// Track the number of bytes read
	var readBytes int64 = 0

	// Scan through the chunk
	for scanner.Scan() { // Read each line in the chunk
		line := scanner.Text()          // Read a line
		lineLen := int64(len(line)) + 1 // +1 for newline character
		readBytes += lineLen

		// Check if any keyword is present in the line
		found := false                     // Initialize found flag to false. false means keyword is not present in the line
		for _, keyword := range keywords { // Iterate through keywords
			if strings.Contains(line, keyword) { // Check for keyword presence
				found = true // Set found flag to true if keyword is found in the line. found true means keyword is present in the lineLen
				break        // Exit loop if keyword found
			}
		}

		// If a keyword is found, process the line further.
		// Print matched lines with regex groups if applicable. Otherwise, print the raw matched line
		if found {
			if regexpCompiled != nil { // If regex pattern is provided in config file
				matches := regexpCompiled.FindStringSubmatch(line) // Find regex matches in the line
				if len(matches) > 1 {                              // 1 is the full match itself
					fmt.Printf("[Worker %d] [DETECTED] Matched Line: %s %v\n", workerID, line, matches[1:]) // Print matched groups
				} else {
					fmt.Printf("[Worker %d] [RAW] Matched Line: %s\n", workerID, line) // Print raw line if no groups matched
				}
			} else {
				fmt.Printf("[Worker %d] [FOUND] Matched Line: %s\n", workerID, line) // Print raw matched line
			}
		}

		// Break the loop if the chunk size is reached
		if size > 0 {
			if readBytes >= size { // Check if chunk size is reached
				break
			}
		}
	}
}

// End of file
