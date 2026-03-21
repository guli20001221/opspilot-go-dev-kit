// Package workflow contains the typed async task promotion layer used when a
// synchronous request must move into a durable workflow path.
//
// The current skeleton only creates in-memory promoted tasks and statuses so
// the runtime can model async promotion before Temporal and task APIs land.
package workflow
