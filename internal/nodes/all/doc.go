// Package all imports every built-in node package for registration side effects.
package all

import (
	_ "github.com/yottaapp/yotta/internal/nodes/ai"
	_ "github.com/yottaapp/yotta/internal/nodes/collection"
	_ "github.com/yottaapp/yotta/internal/nodes/control"
	_ "github.com/yottaapp/yotta/internal/nodes/detect"
	_ "github.com/yottaapp/yotta/internal/nodes/event"
	_ "github.com/yottaapp/yotta/internal/nodes/image"
	_ "github.com/yottaapp/yotta/internal/nodes/input"
	_ "github.com/yottaapp/yotta/internal/nodes/io"
	_ "github.com/yottaapp/yotta/internal/nodes/purefunc"
	_ "github.com/yottaapp/yotta/internal/nodes/random"
	_ "github.com/yottaapp/yotta/internal/nodes/script"
	_ "github.com/yottaapp/yotta/internal/nodes/stopwatch"
	_ "github.com/yottaapp/yotta/internal/nodes/system"
	_ "github.com/yottaapp/yotta/internal/nodes/variable"
	_ "github.com/yottaapp/yotta/internal/nodes/window"
)
