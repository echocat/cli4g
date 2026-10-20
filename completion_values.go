package cli

import (
	"context"
	"slices"

	"github.com/echocat/cli4g/codec"
)

type completionValues struct {
	scan *completionScan
	opts *ParseOpts
}

func (values *completionValues) Flag(ctx context.Context, name string) (codec.CompletionValue, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return codec.CompletionValue{}, false, err
	}
	owner, flag := findInputFlag(values.scan.path, name, 0)
	if flag == nil {
		return codec.CompletionValue{}, false, nil
	}
	if occurrences, ok := values.scan.scopes[owner].Flags[flag.Name()]; ok {
		result := codec.CompletionValue{Source: codec.CompletionCLI}
		for _, occurrence := range occurrences {
			result.Values = append(result.Values, occurrence.Values...)
		}
		return result, true, nil
	}
	for _, variable := range flag.envVars {
		value, exists := values.opts.lookupEnv(ctx, variable)
		if err := ctx.Err(); err != nil {
			return codec.CompletionValue{}, false, err
		}
		if exists {
			return codec.CompletionValue{Source: codec.CompletionEnvironment, Values: []string{value}, Environment: variable}, true, nil
		}
	}
	return completionDefault(ctx, flag.defaultValue, flag.codec)
}

func (values *completionValues) Argument(ctx context.Context, name string) (codec.CompletionValue, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return codec.CompletionValue{}, false, err
	}
	leaf := len(values.scan.path) - 1
	for _, argument := range values.scan.path[leaf].schema.arguments {
		if argument.Name() != name {
			continue
		}
		if tokens, found := values.scan.scopes[leaf].Arguments[name]; found {
			return codec.CompletionValue{Source: codec.CompletionCLI, Values: slices.Clone(tokens)}, true, nil
		}
		return completionDefault(ctx, argument.defaultValue, argument.codec)
	}
	return codec.CompletionValue{}, false, nil
}

func completionDefault(ctx context.Context, source *valueDefault[any], selected codec.Codec[any, any]) (codec.CompletionValue, bool, error) {
	if source == nil {
		return codec.CompletionValue{}, false, nil
	}
	result := codec.CompletionValue{Source: codec.CompletionDefault}
	if source.isString {
		result.Values = slices.Clone(source.strings)
		return result, true, ctx.Err()
	}
	requirement, _, err := resolveDefault(ctx, source, selected, nil)
	if err != nil {
		return result, true, err
	}
	result.Values, err = selected.FormatRequirement(ctx, requirement)
	result.Values = slices.Clone(result.Values)
	if err != nil {
		return result, true, codec.SanitizeError(err, "completion default could not be formatted")
	}
	return result, true, ctx.Err()
}
