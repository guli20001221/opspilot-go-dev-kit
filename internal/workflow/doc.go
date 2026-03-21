// Package workflow contains the typed async task promotion layer used when a
// synchronous request must move into a durable workflow path.
//
// The current skeleton persists task records through a store abstraction so the
// runtime can expose durable task state before full Temporal execution lands.
package workflow
