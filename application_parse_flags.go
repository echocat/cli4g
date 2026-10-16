package cli

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/echocat/cli4g/codec"
)

func scanLongFlag(ctx context.Context, vs []string, pos *int, path []selectedContainer, selectCommands bool) (flagInput, error) {
	name, inline, hasValue := strings.Cut(vs[*pos][2:], "=")
	scope, definition := findInputFlag(path, name, 0)
	if definition == nil {
		return flagInput{}, NewValidationErrorf("unknown flag --%s", name)
	}
	var values []string
	if hasValue {
		values = []string{inline}
	}
	values, err := consumeFlagValues(ctx, vs, pos, values, definition.arity, path[len(path)-1], selectCommands)
	if err != nil {
		return flagInput{}, err
	}
	if err := definition.arity.check(len(values)); err != nil {
		return flagInput{}, NewValidationErrorf("flag --%s: %w", name, err)
	}
	return flagInput{scope: scope, definition: definition, values: values}, nil
}

func scanShortFlags(ctx context.Context, vs []string, pos *int, path []selectedContainer, selectCommands bool) ([]flagInput, bool, error) {
	shorts := vs[*pos][1:]
	var result []flagInput
	for len(shorts) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		short, size := utf8.DecodeRuneInString(shorts)
		shorts = shorts[size:]
		scope, definition := findInputFlag(path, "", short)
		if definition == nil {
			return nil, false, NewValidationErrorf("unknown short flag -%c", short)
		}
		var values []string
		hasInlineValue := strings.HasPrefix(shorts, "=")
		consumesValues := hasInlineValue || definition.arity.min > 0 || definition.arity.max > 1 || definition.arity.max == -1
		if consumesValues {
			if len(shorts) > 0 {
				// Decode only names as Unicode; preserve the value's original bytes.
				// Remove only the explicit separator, preserving later =.
				values = []string{strings.TrimPrefix(shorts, "=")}
			}
			var err error
			values, err = consumeFlagValues(ctx, vs, pos, values, definition.arity, path[len(path)-1], selectCommands)
			if err != nil {
				return nil, false, err
			}
		}
		if err := definition.arity.check(len(values)); err != nil {
			return nil, false, NewValidationErrorf("flag -%c: %w", short, err)
		}
		flag := flagInput{scope: scope, definition: definition, values: values}
		if definition.isHelp {
			requested, err := scanHelpFlag(ctx, flag)
			if err != nil {
				return nil, false, err
			}
			if requested {
				return result, true, ctx.Err()
			}
		}
		result = append(result, flag)
		if consumesValues {
			break
		}
	}
	return result, false, nil
}

// Only the immutable framework bool codec is used during scanning. Other
// values remain undecoded; help stops before options or user codecs are used.
func scanHelpFlag(ctx context.Context, flag flagInput) (bool, error) {
	value, err := flag.definition.codec.ParseRequirement(ctx, codec.UsageFlag, flag.values...)
	if err != nil {
		return false, fmt.Errorf("flag --help: %w", err)
	}
	return value.(bool), nil
}

// Optional single-value flags require an inline value (--verbose=false or
// -v=false). Required values are consumed even when they equal a command name.
// Additional variadic values stop
// at the next option or still-selectable command. -- always ends value consumption.
func consumeFlagValues(ctx context.Context, vs []string, pos *int, values []string, arity valueArity, current selectedContainer, selectCommands bool) ([]string, error) {
	if arity.max == 0 || arity == (valueArity{min: 0, max: 1}) {
		return values, ctx.Err()
	}
	for *pos+1 < len(vs) && (arity.max == -1 || len(values) < arity.max) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		next := vs[*pos+1]
		if next == "--" {
			break
		}
		if len(values) >= arity.min {
			if strings.HasPrefix(next, "-") && next != "-" {
				break
			}
			if selectCommands {
				if _, ok := current.commandByName[next]; ok {
					break
				}
			}
		}
		*pos++
		values = append(values, next)
	}
	return values, ctx.Err()
}

func findInputFlag(path []selectedContainer, name string, short rune) (int, *Flag[any, any]) {
	for i, p := range slices.Backward(path) {
		var definition *Flag[any, any]
		var found bool
		if name != "" {
			definition, found = p.schema.flagByName[name]
		} else {
			definition, found = p.schema.flagByShort[short]
		}
		if found {
			return i, definition
		}
	}
	return 0, nil
}
