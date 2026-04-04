package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/llmeval/pkg/evaluator"
	"github.com/user/llmeval/pkg/llm"
)

type TestCase struct {
	ID       string             `json:"id"`
	Prompt   string             `json:"prompt"`
	Expected string             `json:"expected"`
	EvalType evaluator.EvalType `json:"eval_type"`
}

type TestResult struct {
	TestCase TestCase
	Actual   string
	Pass     bool
	Reason   string
	Error    error
	Duration time.Duration
}

type Runner struct {
	TestModel  string
	LLMClient  *llm.Client
	Evaluator  *evaluator.Evaluator
	MaxWorkers int
}

func NewRunner(client *llm.Client, eval *evaluator.Evaluator, testModel string, concurrency int) *Runner {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Runner{
		TestModel:  testModel,
		LLMClient:  client,
		Evaluator:  eval,
		MaxWorkers: concurrency,
	}
}

func (r *Runner) Run(ctx context.Context, cases []TestCase) []TestResult {
	jobs := make(chan TestCase, len(cases))
	results := make(chan TestResult, len(cases))

	var wg sync.WaitGroup

	for i := 0; i < r.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tc := range jobs {
				results <- r.runSingle(ctx, tc)
			}
		}()
	}

	for _, tc := range cases {
		jobs <- tc
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []TestResult
	for res := range results {
		allResults = append(allResults, res)
	}

	return allResults
}

func (r *Runner) runSingle(ctx context.Context, tc TestCase) TestResult {
	start := time.Now()

	actual, err := r.LLMClient.Generate(ctx, r.TestModel, tc.Prompt, 0.0)
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			TestCase: tc,
			Error:    fmt.Errorf("failed to generate response: %w", err),
			Duration: duration,
		}
	}

	evalRes, err := r.Evaluator.Evaluate(ctx, tc.EvalType, tc.Prompt, tc.Expected, actual)
	if err != nil {
		return TestResult{
			TestCase: tc,
			Actual:   actual,
			Error:    fmt.Errorf("evaluation failed: %w", err),
			Duration: duration,
		}
	}

	return TestResult{
		TestCase: tc,
		Actual:   actual,
		Pass:     evalRes.Pass,
		Reason:   evalRes.Reason,
		Duration: duration,
	}
}
