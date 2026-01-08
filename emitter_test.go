package reorder

import "testing"

func TestEmitterRegistry(t *testing.T) {
	t.Run("all sections have emitters", func(t *testing.T) {
		for section := range ValidSections {
			emitter := getEmitter(section)
			if emitter == nil {
				t.Errorf("no emitter for section %q", section)
			}
		}
	})

	t.Run("unknown section returns nil", func(t *testing.T) {
		emitter := getEmitter("bogus_section")
		if emitter != nil {
			t.Error("expected nil emitter for unknown section")
		}
	})
}

func TestEmittersHandleEmpty(t *testing.T) {
	cat := &categorizedDecls{}
	cfg := DefaultConfig()

	for section := range ValidSections {
		t.Run(section, func(t *testing.T) {
			emitter := getEmitter(section)
			if emitter == nil {
				t.Skip("no emitter")
			}
			// Should not panic on empty categorizedDecls
			decls := emitter(cat, cfg)
			if decls == nil {
				t.Error("emitter returned nil, expected empty slice")
			}
		})
	}
}