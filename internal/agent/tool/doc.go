// Package tool contains the typed tool-execution stage for the agent runtime.
//
// The current skeleton executes deterministic stub tools through the registry,
// distinguishes read-only from approval-gated actions, and returns auditable
// tool results without talking to live external systems.
package tool
