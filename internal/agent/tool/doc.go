// Package tool contains the typed tool-execution stage for the agent runtime.
//
// The current skeleton executes deterministic typed tools through the registry,
// distinguishes read-only from approval-gated actions, and returns auditable
// tool results. When configured, the same tool contracts can cross an HTTP
// adapter boundary without changing the agent-runtime call sites.
package tool
