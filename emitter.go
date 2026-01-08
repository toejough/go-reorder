package reorder

import "github.com/dave/dst"

// sectionEmitter emits declarations for a section from categorized declarations.
type sectionEmitter func(*categorizedDecls) []dst.Decl

// getEmitter returns the emitter for a section name, or nil if unknown.
func getEmitter(section string) sectionEmitter {
	return nil
}