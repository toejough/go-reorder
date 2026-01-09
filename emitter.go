package reorder

import (
	"github.com/dave/dst"

	"github.com/toejough/go-reorder/internal/categorize"
	"github.com/toejough/go-reorder/internal/emit"
)

// sectionEmitter emits declarations for a section from categorized declarations.
type sectionEmitter func(*categorize.CategorizedDecls, *Config) []dst.Decl

// getEmitter returns the emitter for a section name, or nil if unknown.
func getEmitter(section string) sectionEmitter {
	emitter := emit.GetEmitter(section)
	if emitter == nil {
		return nil
	}

	// Wrap the emit.SectionEmitter to use our Config type
	return func(cat *categorize.CategorizedDecls, cfg *Config) []dst.Decl {
		emitCfg := &emit.Config{
			TypeLayout: cfg.Types.TypeLayout,
			EnumLayout: cfg.Types.EnumLayout,
		}

		return emitter(cat, emitCfg)
	}
}
