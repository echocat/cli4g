package main

import (
	"context"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// A real deployment tool could load this inventory from a cluster client.
// The local data keeps both planning and shell suggestions self-contained.
var inventory = map[string]map[string][]string{
	"development": {
		"default":  {"api", "worker"},
		"payments": {"checkout", "billing"},
	},
	"production": {
		"default":  {"api"},
		"payments": {"checkout", "billing"},
		"platform": {"gateway", "worker"},
	},
}

func serviceExists(cluster, namespace, service string) bool {
	return slices.Contains(inventory[cluster][namespace], service)
}

func completeNamespaces(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	cluster, err := completionFlag(ctx, input, "context")
	if err != nil {
		return codec.CompletionResult{}, err
	}
	service := ""
	if values := input.Arguments["service"]; len(values) > 0 {
		service = values[0]
	}
	result := codec.CompletionResult{Handled: true}
	for namespace, services := range inventory[cluster] {
		if err := ctx.Err(); err != nil {
			return codec.CompletionResult{}, err
		}
		if service == "" || slices.Contains(services, service) {
			result.Candidates = append(result.Candidates, codec.CompletionCandidate{
				Value: namespace, Description: "Namespace in " + cluster,
			})
		}
	}
	return result, nil
}

func completeServices(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	cluster, err := completionFlag(ctx, input, "context")
	if err != nil {
		return codec.CompletionResult{}, err
	}
	namespace, err := completionFlag(ctx, input, "namespace")
	if err != nil {
		return codec.CompletionResult{}, err
	}
	result := codec.CompletionResult{Handled: true}
	for _, service := range inventory[cluster][namespace] {
		if err := ctx.Err(); err != nil {
			return codec.CompletionResult{}, err
		}
		result.Candidates = append(result.Candidates, codec.CompletionCandidate{
			Value: service, Description: "Service in " + namespace,
		})
	}
	return result, nil
}

func completionFlag(ctx context.Context, input codec.CompletionInput, name string) (string, error) {
	value, _, err := input.Effective.Flag(ctx, name)
	if err != nil || len(value.Values) == 0 {
		return "", err
	}
	return value.Values[len(value.Values)-1], nil
}
