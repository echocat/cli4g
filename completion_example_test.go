package cli_test

import (
	"context"
	"fmt"

	cli "github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

// A real implementation could ask a Kubernetes client using the same ctx.
func namespaceSuggestions(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	cluster, _, err := input.Effective.Flag(ctx, "context")
	if err != nil {
		return codec.CompletionResult{}, err
	}
	name := "default"
	if len(cluster.Values) > 0 {
		name = cluster.Values[len(cluster.Values)-1]
	}
	// Explicit raw occurrences are also available as input.Flags["context"].
	// input.Arguments and input.Positionals describe the selected command's args.
	return codec.CompletionResult{
		Handled: true,
		Candidates: []codec.CompletionCandidate{
			{Value: name + "-apps", Description: "Application namespace."},
			{Value: name + "-system", Description: "System namespace."},
		},
	}, ctx.Err()
}

func ExampleApplication_Complete() {
	app := cli.NewApplication[struct {
		Context   string `flag:"context" default:"development"`
		Namespace string `flag:"namespace" type:"namespace"`
	}]("tool").WithCompletion(true).
		WithCodec(codec.WithCompletionHandler(codec.NewString(), namespaceSuggestions), "namespace").MustCreate()
	defer func() { _ = app.Close() }()

	// A cursor can precede other dependency flags. No options are initialized.
	result, err := app.Complete(context.Background(), nil, cli.CompletionRequest{
		Words: []string{"--namespace", "pro", "--context", "production"},
		Word:  1, Offset: 3,
	})
	if err != nil {
		panic(err)
	}
	for _, candidate := range result.Candidates {
		fmt.Println(candidate.Value, "—", candidate.Description)
	}
	// Output:
	// production-apps — Application namespace.
	// production-system — System namespace.
}
