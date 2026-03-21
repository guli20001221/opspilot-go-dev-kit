// Package retrieval contains the typed retrieval stage and provenance-bearing
// evidence results used by the agent runtime.
//
// The current skeleton performs deterministic in-memory matching against a
// static evidence catalog so the request and result contracts can stabilize
// before the pgvector-backed implementation lands.
package retrieval
