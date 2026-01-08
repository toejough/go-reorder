package main

import (
	"fmt"
	"os"

	"github.com/toejough/go-reorder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/reorder-demo <input.go> [config.toml]")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	input, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	var cfg *reorder.Config
	if len(os.Args) >= 3 {
		cfg, err = reorder.LoadConfig(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = reorder.DefaultConfig()
	}

	result, err := reorder.SourceWithConfig(string(input), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reordering: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(result)
}
