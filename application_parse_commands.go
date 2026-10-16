package cli

import (
	"context"
	"fmt"
	"strings"
)

// scanTokens uses the compiled arity for both command selection and flag value
// consumption. Codec overrides cannot change inherited arity, so the final
// command path can select its decoder after this single pass.
func (this *Application[O]) scanTokens(ctx context.Context, profiles []string, vs []string) (input parseInput, err error) {
	input = parseInput{positions: make([][]string, 1)}
	// Help and failed scans also retain the final command path's codec overrides.
	defer func() {
		if len(input.scopes) == 0 {
			return
		}
		plan := input.scopes[len(input.scopes)-1].container.base().plan
		for i, source := range plan.schemas {
			selected, selectionErr := selectSchema(source, profiles)
			if selectionErr != nil {
				if err == nil {
					err = fmt.Errorf("selected schema: %w", selectionErr)
				}
				continue
			}
			input.scopes[i].schema = selected
		}
	}()
	for _, profile := range profiles {
		if !this.ContainsProfile(profile) {
			return input, NewValidationErrorf("unknown profile %q; known profiles: %v", profile, this.definition.profiles)
		}
	}
	current, err := selectContainer(this, nil, profiles)
	input.scopes = append(input.scopes, current)
	if err != nil {
		return input, NewValidationErrorf("%s: selected profiles %v: %v", this.Name(), profiles, err)
	}
	path := input.scopes
	namePath := this.Name()
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
			if command, ok := current.commandByName[token]; ok {
				input.commands = append(input.commands, command)
				input.positions = append(input.positions, nil)
				current, err = selectContainer(command, path, profiles)
				namePath += " " + command.Name()
				path = append(path, current)
				input.scopes = path
				if err != nil {
					return input, NewValidationErrorf("%s: selected profiles %v: %v", namePath, profiles, err)
				}
				continue
			}
		}
		input.positions[active] = append(input.positions[active], token)
	}
	return input, ctx.Err()
}
