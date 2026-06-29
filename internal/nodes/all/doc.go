// Package all imports every built-in node package for registration side effects.
package all

import (
	_ "yotta/internal/nodes/ai"
	_ "yotta/internal/nodes/collection"
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"
	_ "yotta/internal/nodes/image"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/random"
	_ "yotta/internal/nodes/script"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
	_ "yotta/internal/nodes/window"
)
