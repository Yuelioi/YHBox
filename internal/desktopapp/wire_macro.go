package desktopapp

import (
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/macro"
)

// newMacroService exposes versioned, editable atomic macros through the shared
// asset store. Precise InputClip recordings deliberately remain a separate
// service and asset kind.
func newMacroService(store *asset.Store, emit ...func(name string, data any)) *macro.Service {
	return macro.NewService(store, emit...)
}
