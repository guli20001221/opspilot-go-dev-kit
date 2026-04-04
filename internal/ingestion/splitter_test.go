package ingestion

import "testing"

func TestSplitterBasicParagraphs(t *testing.T) {
	s := &SentenceSplitter{}
	sentences := s.Split("First paragraph here.\n\nSecond paragraph here.")
	if len(sentences) != 2 {
		t.Fatalf("len = %d, want 2", len(sentences))
	}
	if sentences[0].Text != "First paragraph here." {
		t.Fatalf("sentences[0] = %q", sentences[0].Text)
	}
	if sentences[1].Text != "Second paragraph here." {
		t.Fatalf("sentences[1] = %q", sentences[1].Text)
	}
}

func TestSplitterSentenceBoundaries(t *testing.T) {
	s := &SentenceSplitter{}
	sentences := s.Split("First sentence. Second sentence? Third sentence!")
	if len(sentences) != 3 {
		t.Fatalf("len = %d, want 3", len(sentences))
	}
	if sentences[0].Index != 0 || sentences[1].Index != 1 || sentences[2].Index != 2 {
		t.Fatalf("indices = %d,%d,%d", sentences[0].Index, sentences[1].Index, sentences[2].Index)
	}
}

func TestSplitterEmpty(t *testing.T) {
	s := &SentenceSplitter{}
	if sentences := s.Split(""); len(sentences) != 0 {
		t.Fatalf("len = %d, want 0", len(sentences))
	}
}

func TestSplitterShortFragmentsSkipped(t *testing.T) {
	s := &SentenceSplitter{MinLen: 10}
	sentences := s.Split("Hi. This is a longer sentence that passes the minimum.")
	if len(sentences) != 1 {
		t.Fatalf("len = %d, want 1 (short fragment skipped)", len(sentences))
	}
}
