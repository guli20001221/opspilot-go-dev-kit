package ingestion

import "strings"

// SentenceSplitter splits document text into sentences.
type SentenceSplitter struct {
	MinLen int // minimum sentence length in characters; default 10
}

// Split splits text into sentences by paragraph and sentence boundaries.
func (s *SentenceSplitter) Split(text string) []Sentence {
	minLen := s.MinLen
	if minLen <= 0 {
		minLen = 10
	}

	var sentences []Sentence
	paragraphs := strings.Split(text, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) < minLen {
			continue
		}
		parts := splitSentences(para)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if len(part) < minLen {
				continue
			}
			sentences = append(sentences, Sentence{
				Text:  part,
				Index: len(sentences),
			})
		}
	}
	return sentences
}

func splitSentences(text string) []string {
	var result []string
	current := strings.Builder{}

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])
		if isSentenceEnd(runes, i) {
			result = append(result, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func isSentenceEnd(runes []rune, i int) bool {
	ch := runes[i]
	if ch != '.' && ch != '?' && ch != '!' {
		return false
	}
	// Must be followed by space, newline, or end of text
	if i+1 >= len(runes) {
		return true
	}
	next := runes[i+1]
	return next == ' ' || next == '\n' || next == '\r' || next == '\t'
}
