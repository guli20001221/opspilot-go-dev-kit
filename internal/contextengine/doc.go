// Package contextengine assembles typed context blocks for planner, retrieval,
// and critic stages.
//
// The current skeleton keeps assembly deterministic and explicit by building a
// small set of blocks from request metadata, recent turns, session summary, and
// task scratchpad content, while also returning an assembly log with included
// and dropped blocks.
package contextengine
