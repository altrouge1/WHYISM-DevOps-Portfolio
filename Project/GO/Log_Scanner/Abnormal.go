/*
Abnormal.go
   A Go program to scan large log files for abnormal patterns based on user-defined keywords and regex patterns.
   It processes the log file in parallel using goroutines for efficiency.
   The program reads a configuration file (config.json) to get the target log file path,
   keywords to search for, and an optional regex pattern to extract additional information from matched log lines.
*/

package main

/*
	bufio: Provides buffered I/O operations for efficient reading of files.
	fmt: For formatted I/O operations like printing to console.
	os: Provides functions to interact with the operating system, such as file handling.
	regexp: For compiling and using regular expressions.
	runtime: To get information about the Go runtime, such as number of CPU cores.
	strings: For string manipulation functions.
	sync: Provides synchronization primitives like WaitGroup for managing goroutines.
*/
import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// Main function to execute the log scanner program
func runAbnormalMode(config *Config) {
	file, err := os.Open(config.TargetLogPath)
	// Handle file open error
	if err != nil {
		fmt.Printf("[ERROR] Failed to open file: %v\n", err)
		return
	}
	defer file.Close()
	// Get file size for chunking the log file among workers
	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("[ERROR] Failed to get file info: %v\n", err)
		return
	}
	// Calculate number of workers and chunk size based on file size and CPU cores available
	fileSize := stat.Size()
	numWorkers := runtime.NumCPU()
	chunkSize := fileSize / int64(numWorkers)
	var WaitGroup sync.WaitGroup

	fmt.Printf("[SYSTEM] Parallel Scan: %d Workers / %d GB\n", numWorkers, fileSize/1024/1024/1024)

	var regex *regexp.Regexp
	// Handle regex compilation error
	if config.LogPattern != "" {
		regex = regexp.MustCompile(config.LogPattern)
	}
	// Launch goroutines for each worker to process chunks of the log file in parallel
	for Workers := 0; Workers < numWorkers; Workers++ {
		start := int64(Workers) * chunkSize
		size := chunkSize
		if Workers == numWorkers-1 {
			size = fileSize - start
		}
		// Increment WaitGroup counter and launch goroutine for processing the chunk
		WaitGroup.Add(1)
		go func(id int, start, size int64) {
			defer WaitGroup.Done()
			processChunk(config.TargetLogPath, start, size, config.Keywords, regex, id)
		}(Workers, start, size)
	}

	WaitGroup.Wait()
	fmt.Println("[SYSTEM] Parallel Scan Complete.")
}

// Function to process a chunk of the log file by a worker goroutine
func processChunk(filePath string, start, size int64, keywords []string, regex *regexp.Regexp, workerID int) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("[Worker %d] [ERROR] Failed to open file: %v\n", workerID, err)
		return
	}
	defer file.Close()
	// Seek to the start position of the chunk in the file
	if _, err := file.Seek(start, 0); err != nil {
		fmt.Printf("[Worker %d] [ERROR] Failed to seek file: %v\n", workerID, err)
		return
	}

	reader := bufio.NewReader(file)
	// If not at the beginning, read until the end of the current line to avoid partial lines
	if start > 0 {
		_, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[Worker %d] [ERROR] Failed to read line: %v\n", workerID, err)
			return
		}
	}
	// Adjust size to account for the skipped partial line
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 64*1024)   // 64KB buffer for scanner
	scanner.Buffer(buf, 1024*1024) // Max token size 1MB

	var readBytes int64 = 0 // Track bytes read in this chunk
	// Scan through each line in the chunk and check for keywords and patterns
	for scanner.Scan() {
		line := scanner.Text()
		readBytes += int64(len(line)) + 1
		// Check for presence of any keyword in the current line
		found := false
		for _, keyword := range keywords {
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
					fmt.Printf("[Worker %d] [DETECTED] %v\n", workerID, matches[1:])
				} else {
					fmt.Printf("[Worker %d] [RAW] %s\n", workerID, line)
				}
			} else {
				fmt.Printf("[Worker %d] [FOUND] %s\n", workerID, line)
			}
		}
		// Break if the read bytes exceed the assigned chunk size
		if size > 0 && readBytes >= size {
			break
		}
	}
}
