package retrieval

// ReorderLostInTheMiddle reorders evidence blocks so the most relevant items
// appear at the beginning and end of the list, mitigating the "lost in the
// middle" problem where LLMs underweight information in the center of long
// contexts (Liu et al., NeurIPS 2023).
//
// Input must be sorted by relevance (highest first). Output alternates:
// rank 1 → start, rank 2 → end, rank 3 → start, rank 4 → end, etc.
func ReorderLostInTheMiddle(blocks []EvidenceBlock) []EvidenceBlock {
	if len(blocks) <= 2 {
		return blocks
	}

	result := make([]EvidenceBlock, len(blocks))
	left, right := 0, len(blocks)-1

	for i, block := range blocks {
		if i%2 == 0 {
			result[left] = block
			left++
		} else {
			result[right] = block
			right--
		}
	}

	return result
}
