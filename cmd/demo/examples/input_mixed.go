//go:build ignore

package mixedexample

func Helper() {}

const Version = "1.0"

type Config struct {
	Name string
}

var DefaultConfig = Config{Name: "default"}

func NewConfig() *Config {
	return &Config{}
}
