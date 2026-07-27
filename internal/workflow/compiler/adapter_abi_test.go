package compiler

import "github.com/yottaapp/yotta/internal/nodeadapter"

// Compiler tests use the real adapter ABI without restoring production
// re-exports from this package.
type Adapter = nodeadapter.Adapter
type AdapterAction = nodeadapter.AdapterAction
type AdapterResult = nodeadapter.AdapterResult
type InstalledAdapter = nodeadapter.InstalledAdapter
type Invocation = nodeadapter.Invocation
type NodeFailure = nodeadapter.NodeFailure
type RoutedFailure = nodeadapter.RoutedFailure
type SignalTrigger = nodeadapter.SignalTrigger
