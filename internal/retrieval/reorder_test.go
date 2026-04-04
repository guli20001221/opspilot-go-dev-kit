package retrieval

import "testing"

func TestReorderLostInTheMiddle(t *testing.T) {
	blocks := []EvidenceBlock{
		{EvidenceID: "a", Score: 0.9},
		{EvidenceID: "b", Score: 0.8},
		{EvidenceID: "c", Score: 0.7},
		{EvidenceID: "d", Score: 0.6},
		{EvidenceID: "e", Score: 0.5},
	}

	result := ReorderLostInTheMiddle(blocks)

	// Expect: a(start), c(start), e(start), d(end), b(end)
	// Highest at position 0, second-highest at last position
	if result[0].EvidenceID != "a" {
		t.Fatalf("result[0] = %q, want a (rank 1 at start)", result[0].EvidenceID)
	}
	if result[len(result)-1].EvidenceID != "b" {
		t.Fatalf("result[last] = %q, want b (rank 2 at end)", result[len(result)-1].EvidenceID)
	}
}

func TestReorderLostInTheMiddleSmall(t *testing.T) {
	blocks := []EvidenceBlock{{EvidenceID: "a"}, {EvidenceID: "b"}}
	result := ReorderLostInTheMiddle(blocks)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestReorderLostInTheMiddleEmpty(t *testing.T) {
	result := ReorderLostInTheMiddle(nil)
	if result != nil {
		t.Fatalf("result = %v, want nil", result)
	}
}
