package retrieval

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const (
	defaultTopK = 8
	maxTopK     = 12
)

// Service performs deterministic in-memory retrieval over a static catalog.
type Service struct {
	catalog []EvidenceBlock
}

// NewService constructs a retrieval service over the provided evidence catalog.
func NewService(catalog []EvidenceBlock) *Service {
	cloned := make([]EvidenceBlock, len(catalog))
	copy(cloned, catalog)
	return &Service{catalog: cloned}
}

// Search executes a structured retrieval request and returns provenance-rich evidence.
func (s *Service) Search(_ context.Context, req RetrievalRequest) (RetrievalResult, error) {
	queryUsed := strings.TrimSpace(req.RewrittenQuery)
	if queryUsed == "" {
		queryUsed = strings.TrimSpace(req.QueryText)
	}

	ranked := make([]EvidenceBlock, 0, len(s.catalog))
	for _, candidate := range s.catalog {
		if candidate.TenantID != req.TenantID {
			continue
		}

		score := scoreCandidate(queryUsed, candidate)
		if score <= 0 {
			continue
		}

		candidate.Score = score
		if req.UseRerank {
			candidate.RerankScore = score
		}
		ranked = append(ranked, candidate)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].EvidenceID < ranked[j].EvidenceID
		}
		return ranked[i].Score > ranked[j].Score
	})

	topK := normalizeTopK(req.TopK)
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	for i := range ranked {
		if ranked[i].CitationLabel == "" {
			ranked[i].CitationLabel = fmt.Sprintf("[%d]", i+1)
		}
		if ranked[i].PermissionsScope == "" {
			ranked[i].PermissionsScope = "tenant:" + req.TenantID
		}
	}

	result := RetrievalResult{
		RequestID:      req.RequestID,
		PlanID:         req.PlanID,
		QueryUsed:      queryUsed,
		EvidenceBlocks: ranked,
	}
	if len(ranked) > 0 {
		result.CoverageScore = 1
	} else if queryUsed != "" {
		result.MissingQuestions = []string{queryUsed}
	}

	return result, nil
}

func normalizeTopK(topK int) int {
	if topK <= 0 {
		return defaultTopK
	}
	if topK > maxTopK {
		return maxTopK
	}

	return topK
}

func scoreCandidate(query string, candidate EvidenceBlock) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}

	haystacks := []string{
		strings.ToLower(candidate.SourceTitle),
		strings.ToLower(candidate.Snippet),
	}
	score := 0.0
	for _, term := range strings.Fields(query) {
		if term == "" {
			continue
		}
		for _, haystack := range haystacks {
			if strings.Contains(haystack, term) {
				score += 1
				break
			}
		}
	}

	return score
}
