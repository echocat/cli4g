package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	cli "github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
)

var variant = "initial"

var completionCommand = "completion"
var completionRequestCommand = "__complete"

type namespaceCodec struct{ codec.Codec[string, string] }

func (source namespaceCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, source)
}

func (source namespaceCodec) Complete(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	cluster, _, err := input.Effective.Flag(ctx, "context")
	if err != nil {
		return codec.CompletionResult{}, err
	}
	name := "production"
	if len(cluster.Values) > 0 {
		name = cluster.Values[0] + "-namespace"
	}
	return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: name, Description: "Namespace in the selected cluster."}}}, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "__quote-stdin" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(string(data)); err != nil {
			panic(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "__echo-args" {
		if err := json.NewEncoder(os.Stdout).Encode(os.Args[2:]); err != nil {
			panic(err)
		}
		return
	}
	app := cli.NewApplication[struct {
		Context   string   `flag:"context" short:"c"`
		Namespace string   `flag:"namespace" short:"n" type:"namespace"`
		Format    string   `flag:"format" enum:"json|yaml|über|two words"`
		Input     *os.File `flag:"input"`
		Selector  string   `flag:"selector" type:"selector"`
		Ordered   string   `flag:"ordered" type:"ordered"`
	}]("cli4g-completion-test").
		WithCompletion(true).
		WithCompletionCommand(cli.NewCompletionCommand().WithName(completionCommand)).
		WithCompletionRequestCommandName(completionRequestCommand).
		WithCodec(namespaceCodec{codec.NewString()}, "namespace").
		WithCodec(codec.WithCompletionHandler(codec.NewString(), func(ctx context.Context, _ codec.CompletionInput) (codec.CompletionResult, error) {
			return codec.CompletionResult{Handled: true, NoSpace: true, Candidates: []codec.CompletionCandidate{{Value: "env="}}}, ctx.Err()
		}), "selector").
		WithCodec(codec.WithCompletionHandler(codec.NewString(), func(ctx context.Context, _ codec.CompletionInput) (codec.CompletionResult, error) {
			return codec.CompletionResult{Handled: true, KeepOrder: true, Candidates: []codec.CompletionCandidate{{Value: "zulu"}, {Value: "alpha"}}}, ctx.Err()
		}), "ordered").
		WithFlag(cli.NewFlag(codec.NewBool(), variant)).
		WithCommand(cli.NewCommand[struct {
			Resource string `argument:"resource" enum:"pods|services"`
		}]("get")).
		WithInitializeHandler(func(context.Context, struct {
			Context   string   `flag:"context" short:"c"`
			Namespace string   `flag:"namespace" short:"n" type:"namespace"`
			Format    string   `flag:"format" enum:"json|yaml|über|two words"`
			Input     *os.File `flag:"input"`
			Selector  string   `flag:"selector" type:"selector"`
			Ordered   string   `flag:"ordered" type:"ordered"`
		}) error {
			if path := os.Getenv("CLI4G_COMPLETION_INIT_MARKER"); path != "" {
				file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err != nil {
					return err
				}
				_, writeErr := file.WriteString("initialize\n")
				closeErr := file.Close()
				if writeErr != nil {
					return writeErr
				}
				return closeErr
			}
			panic("completion must skip initialization")
		})
	app.MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}
