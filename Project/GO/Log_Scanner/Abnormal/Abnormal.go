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

// Config holds the configuration for the log scanner
type Config struct {
	TargetLogPath string   `json:"target_log_path"`
	Keywords      []string `json:"keywords"`
	LogPattern    string   `json:"log_pattern"`
}

func main() {
	// Load configuration file
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
		os.Exit(1)
	}
	// log file Path Finder
	file, err := os.Open(config.TargetLogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file information
	stat, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stat Error: %v\n", err)
		os.Exit(1)
	}

	// Read Abnormaly size log file.
	fileSize := stat.Size()
	numworkers := runtime.NumCPU()
	chunkSize := fileSize / int64(numworkers)

	// Initialize WaitGroup
	var wg sync.WaitGroup

	// Print system information & Log file size & Workers
	fmt.Printf("[SYSTEM] Workers: %d | File Size: %d GB\n", numworkers, fileSize/(1024*1024*1024))

	// Compile regex pattern if provided in config file
	var regexpCompiled *regexp.Regexp
	if config.LogPattern != "" {
		regexpCompiled = regexp.MustCompile(config.LogPattern)
	}

	// Start workers to process log file chunks concurrently
	for worker := 0; worker < numworkers; worker++ {
		start := int64(worker) * chunkSize
		size := chunkSize
		if worker == numworkers-1 {
			size = fileSize - start
		}
		wg.Add(1)
		go func(id int, start, size int64) {
			defer wg.Done()
			processChunk(config.TargetLogPath, start, size, config.Keywords, regexpCompiled, id)
		}(worker, start, size)
	}

	// Wait for all workers to finish
	wg.Wait()
	fmt.Println("[SYSTEM] Parallel Log scanning completed.")
}

// Load configuration from JSON file
func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode JSON configuration
	var configFile Config
	if err := json.NewDecoder(file).Decode(&configFile); err != nil {
		return nil, err
	}
	return &configFile, nil
}

// Process a chunk of the log file concurrently with goroutines
func processChunk(path string, start, size int64, keywords []string, regexpCompiled *regexp.Regexp, workerID int) {
	file, err := os.Open(path)

	// Handle file open error
	if err != nil {
		fmt.Fprintf(os.Stderr, "Worker %d: Error opening log file: %v\n", workerID, err)
		return
	}
	defer file.Close()

	// Seek to the start of the chunk & handle seek error
	if _, err := file.Seek(start, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Worker %d: Error seeking log file: %v\n", workerID, err)
		return
	}

	// Create a buffered reader
	reader := bufio.NewReader(file)

	if start > 0 {
		_, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Worker %d: Error reading log file: %v\n", workerID, err)
			return
		}
	}

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024) // 64KB buffer
	scanner.Buffer(buf, 1024*1024)  // 1MB max token size

	// Track the number of bytes read
	var readBytes int64 = 0

	// Scan through the chunk
	for scanner.Scan() {
		line := scanner.Text()          // Read a line
		lineLen := int64(len(line)) + 1 // +1 for newline character
		readBytes += lineLen

		found := false
		for _, keyword := range keywords {
			if strings.Contains(line, keyword) {
				found = true
				break
			}
		}

		if found {
			if regexpCompiled != nil {
				matches := regexpCompiled.FindStringSubmatch(line)
				if len(matches) > 1 { // 1 is the full match
					fmt.Printf("[Worker %d] [DETECTED] Matched Line: %s %v\n", workerID, line, matches[1:])
				} else {
					fmt.Printf("[Worker %d] [RAW] Matched Line: %s\n", workerID, line)
				}
			} else {
				fmt.Printf("[Worker %d] [FOUND] Matched Line: %s\n", workerID, line)
			}

			if readBytes >= size {
				break
			}
		}
	}
}
