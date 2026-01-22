/*
   GO LOG SCANNER : OPS TOOL
   Made by SEONGEUN KIM
   GitHub: https://github.com/altrouge1/WHYISM-DevOps-Portfolio
   Date: 2026-01-22
   Copyright (c) 2024 SEONGEUN KIM. All rights reserved.
*/
/*
	Active Logic Sequence:
	1. Load configuration from config.json
	2. Display user menu for mode selection (Normal or Abnormal)
	3. Prompt for target log file path (with option to change)
	4. Validate existence of the target log file
	5. Execute selected mode:
	   - Normal Mode: Sequentially scan the log file for keywords
	   - Abnormal Mode: Concurrently scan the log file using goroutines
	6. Output results to console
*/
/*
Normal.go
   A Go program to scan large log files for specific keywords in a sequential manner.
   It reads the log file line by line and checks for the presence of user-defined keywords.
   The program reads a configuration file (config.json) to get the target log file path and keywords to search for.

Abnormal.go
   A Go program to scan large log files for abnormal patterns based on user-defined keywords and regex patterns.
   It processes the log file in parallel using goroutines for efficiency.
   The program reads a configuration file (config.json) to get the target log file path,
   keywords to search for, and an optional regex pattern to extract additional information from matched log lines.
*/
package main

// Main function to execute the log scanner program in normal (sequential) mode
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	/*
		bufio: Provides buffered I/O operations for efficient reading of files.
		fmt: For formatted I/O operations like printing to console.
		os: Provides functions to interact with the operating system, such as file handling.
		encoding/json: For parsing JSON configuration files.
		strings: For string manipulation functions.
	*/)

// Config structure to hold configuration parameters from config.json
type Config struct {
	TargetLogPath string   `json:"target_log_path"`
	Keywords      []string `json:"keywords"`
	LogPattern    string   `json:"log_pattern"`
}

// Function to run the log scanner in normal (sequential) mode
func main() {
	cfg, err := loadConfig("config.json")
	// Handle missing or malformed config file
	if err != nil {
		fmt.Println("[WARN] Config load failed. now using empty config:")
		cfg = &Config{}
		return
	}
	// User Interaction
	reader := bufio.NewReader(os.Stdin) // Standard input reader

	fmt.Println("========================================")
	fmt.Println("      GO LOG SCANNER : OPS TOOL         ")
	fmt.Println("========================================")
	fmt.Println("[1] Normal Mode   (Sequential)")
	fmt.Println("[2] Abnormal Mode (Parallel)")
	fmt.Println("[Q] Quit")
	fmt.Println("----------------------------------------")
	fmt.Print(">> Select Mode: ")

	modeInput, _ := reader.ReadString('\n')                   // Read user input for mode selection
	modeInput = strings.TrimSpace(strings.ToUpper(modeInput)) // Clean input
	// Exit if user chooses to quit
	if modeInput == "Q" {
		fmt.Println("Exiting...")
		return
	}
	// Prompt for target log file path with current path as default
	// Display current target log path from config
	defaultPath := cfg.TargetLogPath
	if defaultPath == "" {
		defaultPath = "(None)"
	}

	fmt.Printf("\n[Target Log Path]\nCurrent: %s\n", defaultPath)
	fmt.Print(">> Press ENTER to use current, or type NEW PATH: ")
	// Read user input for log file path change
	pathInput, _ := reader.ReadString('\n')
	pathInput = strings.TrimSpace(pathInput)
	// Update config if user provided a new path
	if pathInput != "" {
		cfg.TargetLogPath = pathInput
	}
	// Verify the existence of the target log file before proceeding with scanning operations
	if _, err := os.Stat(cfg.TargetLogPath); os.IsNotExist(err) {
		fmt.Printf("\n[ERROR] File not found: %s\n", cfg.TargetLogPath)
		return
	}
	// Execute the selected mode based on user input
	fmt.Println("\n----------------------------------------")
	switch modeInput {
	case "1": // Normal Mode
		if err := runNormalMode(cfg); err != nil {
			fmt.Printf("[FAIL] Normal Mode Error: %v\n", err)
		}
	case "2": // Abnormal Mode
		runAbnormalMode(cfg)
	default:
		fmt.Println("[ERROR] Invalid Selection.")
	}
}

// Function to load configuration from a JSON file
func loadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	// Handle file open error
	if err != nil {
		fmt.Printf("[WARN] Could not open config file: %v\n", err)
		return nil, err
	}
	defer file.Close()

	var cfg Config // Configuration structure
	// Decode JSON into Config struct
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		fmt.Printf("[WARN] Could not parse config file: %v\n", err)
		return nil, err
	}
	return &cfg, nil
}

// End of main.go
