package cli

import (
	"context"
	"strings"
)

// scanTokens uses the compiled arity for both command selection and flag value
// consumption. Codec overrides cannot change inherited arity, so the final
// command path can select its decoder after this single pass.
func (this *Application[O]) scanTokens(ctx context.Context, vs []string) (parseInput, error) {
	input := parseInput{positions: make([][]string, 1)}
	var current Container = this
	path := []Container{this}
	endOfOptions := false
	for i := 0; i < len(vs); i++ {
		if err := ctx.Err(); err != nil {
			return input, err
		}
		token := vs[i]
		if !endOfOptions && token == "--" {
			endOfOptions = true
			continue
		}
		if !endOfOptions && strings.HasPrefix(token, "--") {
			flag, err := scanLongFlag(ctx, vs, &i, path, len(input.positions[len(path)-1]) == 0)
			if err != nil {
				return input, err
			}
			if flag.definition.isHelp {
				requested, err := scanHelpFlag(ctx, flag)
				if err != nil {
					return input, err
				}
				if requested {
					input.helpRequested = true
					return input, ctx.Err()
				}
			}
			input.appendFlags(vs[i+1:], flag)
			continue
		}
		if !endOfOptions && strings.HasPrefix(token, "-") && token != "-" {
			flags, requested, err := scanShortFlags(ctx, vs, &i, path, len(input.positions[len(path)-1]) == 0)
			if err != nil {
				return input, err
			}
			if requested {
				input.helpRequested = true
				return input, ctx.Err()
			}
			input.appendFlags(vs[i+1:], flags...)
			continue
		}
		active := len(path) - 1
		// After the first positional value, command names in that scope are
		// ordinary values. Flags may still be interspersed with these values.
		if !endOfOptions && len(input.positions[active]) == 0 {
			if command, ok := current.Command(token); ok {
				input.commands = append(input.commands, command)
				input.positions = append(input.positions, nil)
				path = append(path, command)
				current = command
				continue
			}
		}
		input.positions[active] = append(input.positions[active], token)
	}
	return input, ctx.Err()
}
