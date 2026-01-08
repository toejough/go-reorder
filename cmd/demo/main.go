package main

import (
	"fmt"
	"os"

	"github.com/toejough/go-reorder"
)

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	if path == "" {
		fmt.Println("Usage: go run ./cmd/demo <config.toml>")
		fmt.Println("       go run ./cmd/demo (no args = show defaults)")
		fmt.Println()
		path = "/nonexistent" // triggers defaults
	}

	cfg, err := reorder.LoadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mode: %s\n\n", cfg.Behavior.Mode)
	fmt.Printf("Sections (%d):\n", len(cfg.Sections.Order))
	for i, s := range cfg.Sections.Order {
		fmt.Printf("  %2d. %s\n", i+1, s)
	}
}