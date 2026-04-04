package retrieval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"opspilot-go/internal/llm"
)

// RAGASMetrics contains the computed RAGAS evaluation scores.
type RAGASMetrics struct {
	Faithfulness     float64 `json:"faithfulness"`      // Is the answer grounded in the context? [0,1]
	AnswerRelevancy  float64 `json:"answer_relevancy"`  // Does the answer address the question? [0,1]
	ContextPrecision float64 `json:"context_precision"` // Are retrieved docs relevant? [0,1]
	OverallScore     float64 `json:"overall_score"`     // Harmonic mean of the three metrics [0,1]
}

// RAGASInput is the input for RAGAS evaluation.
type RAGASInput struct {
	Query       string
	Answer      string
	Contexts    []string // retrieved passages used for generation
	GroundTruth string   // optional reference answer for precision
}

// RAGASEvaluator computes RAGAS-style metrics using LLM-as-judge.
type RAGASEvaluator struct {
	provider llm.Provider
	timeout  time.Duration
}

// NewRAGASEvaluator constructs the evaluator.
func NewRAGASEvaluator(provider llm.Provider) *RAGASEvaluator {
	return &RAGASEvaluator{
		provider: provider,
		timeout:  15 * time.Second,
	}
}

// Evaluate computes all RAGAS metrics for the given input.
func (e *RAGASEvaluator) Evaluate(ctx context.Context, input RAGASInput) (RAGASMetrics, error) {
	if e.provider == nil {
		return RAGASMetrics{}, fmt.Errorf("ragas evaluator requires an LLM provider")
	}

	var errs []error

	faithfulness, faithErr := e.scoreFaithfulness(ctx, input)
	if faithErr != nil {
		slog.Warn("ragas faithfulness failed", slog.Any("error", faithErr))
		errs = append(errs, fmt.Errorf("faithfulness: %w", faithErr))
	}

	relevancy, relErr := e.scoreAnswerRelevancy(ctx, input)
	if relErr != nil {
		slog.Warn("ragas answer relevancy failed", slog.Any("error", relErr))
		errs = append(errs, fmt.Errorf("answer_relevancy: %w", relErr))
	}

	precision, precErr := e.scoreContextPrecision(ctx, input)
	if precErr != nil {
		slog.Warn("ragas context precision failed", slog.Any("error", precErr))
		errs = append(errs, fmt.Errorf("context_precision: %w", precErr))
	}

	overall := harmonicMean(faithfulness, relevancy, precision)

	return RAGASMetrics{
		Faithfulness:     faithfulness,
		AnswerRelevancy:  relevancy,
		ContextPrecision: precision,
		OverallScore:     overall,
	}, errors.Join(errs...)
}

const faithfulnessPrompt = `You are a faithfulness evaluator. Given a context and an answer, determine what fraction of the claims in the answer are supported by the context.

Score from 0.0 to 1.0:
- 1.0 = every claim in the answer is directly supported by the context
- 0.5 = about half the claims are supported
- 0.0 = no claims are supported by the context

Output ONLY a decimal number between 0.0 and 1.0.`

func (e *RAGASEvaluator) scoreFaithfulness(ctx context.Context, input RAGASInput) (float64, error) {
	callCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	contextText := strings.Join(input.Contexts, "\n---\n")
	resp, err := e.provider.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt: faithfulnessPrompt,
		Messages: []llm.Message{{
			Role:    "user",
			Content: fmt.Sprintf("Context:\n%s\n\nAnswer:\n%s", contextText, input.Answer),
		}},
		MaxTokens: 8,
	})
	if err != nil {
		return 0, err
	}
	return parseScore01(resp.Content), nil
}

const answerRelevancyPrompt = `You are an answer relevancy evaluator. Given a question and an answer, determine how well the answer addresses the question.

Score from 0.0 to 1.0:
- 1.0 = the answer fully and directly addresses the question
- 0.5 = the answer partially addresses the question
- 0.0 = the answer is completely unrelated to the question

Output ONLY a decimal number between 0.0 and 1.0.`

func (e *RAGASEvaluator) scoreAnswerRelevancy(ctx context.Context, input RAGASInput) (float64, error) {
	callCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	resp, err := e.provider.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt: answerRelevancyPrompt,
		Messages: []llm.Message{{
			Role:    "user",
			Content: fmt.Sprintf("Question:\n%s\n\nAnswer:\n%s", input.Query, input.Answer),
		}},
		MaxTokens: 8,
	})
	if err != nil {
		return 0, err
	}
	return parseScore01(resp.Content), nil
}

const contextPrecisionPrompt = `You are a context precision evaluator. Given a question and a list of retrieved context passages, determine what fraction of the passages are relevant to answering the question.

Score from 0.0 to 1.0:
- 1.0 = all passages are relevant to the question
- 0.5 = about half the passages are relevant
- 0.0 = no passages are relevant

Output ONLY a decimal number between 0.0 and 1.0.`

func (e *RAGASEvaluator) scoreContextPrecision(ctx context.Context, input RAGASInput) (float64, error) {
	callCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	contextText := strings.Join(input.Contexts, "\n---\n")
	resp, err := e.provider.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt: contextPrecisionPrompt,
		Messages: []llm.Message{{
			Role:    "user",
			Content: fmt.Sprintf("Question:\n%s\n\nContext passages:\n%s", input.Query, contextText),
		}},
		MaxTokens: 8,
	})
	if err != nil {
		return 0, err
	}
	return parseScore01(resp.Content), nil
}

func parseScore01(content string) float64 {
	trimmed := strings.TrimSpace(content)
	score, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0.5 // neutral on parse failure
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func harmonicMean(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sumInverse float64
	for _, v := range values {
		if v <= 0 {
			return 0
		}
		sumInverse += 1.0 / v
	}
	return float64(len(values)) / sumInverse
}
