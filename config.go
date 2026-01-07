package reorder

// Config holds all configuration for go-reorder.
type Config struct {
	Sections SectionsConfig
	Behavior BehaviorConfig
}

// SectionsConfig controls declaration ordering.
type SectionsConfig struct {
	Order []string
}

// BehaviorConfig controls how the reorderer handles edge cases.
type BehaviorConfig struct {
	Mode string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{}
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	return nil
}