package main

import (
	"fmt"

	"github.com/toejough/go-reorder"
	"github.com/toejough/targ"
)

// Demo loads and displays a go-reorder config file.
type Demo struct {
	Config string `targ:"positional,desc=Path to config.toml (omit for defaults)"`
}

func (d *Demo) Description() string {
	return "Load and display a go-reorder config file"
}

func (d *Demo) Run() error {
	path := d.Config
	if path == "" {
		path = "/nonexistent" // triggers defaults
		fmt.Println("No config specified, showing defaults")
	}

	cfg, err := reorder.LoadConfig(path)
	if err != nil {
		return err
	}

	fmt.Printf("Mode: %s\n\n", cfg.Behavior.Mode)
	fmt.Printf("Sections (%d):\n", len(cfg.Sections.Order))
	for i, s := range cfg.Sections.Order {
		fmt.Printf("  %2d. %s\n", i+1, s)
	}
	return nil
}

func main() {
	targ.Run(&Demo{})
}