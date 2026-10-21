package cli

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/echocat/cli4g/codec"
)

type completionScan struct {
	request     CompletionRequest
	profiles    selectedProfiles
	path        []selectedContainer
	scopes      []codec.CompletionScope
	positions   [][]int
	target      codec.CompletionTarget
	valuePrefix string
	commandSite bool
}

func (scan *completionScan) addScope(selected selectedContainer) {
	scan.path = append(scan.path, selected)
	var names []string
	if len(scan.scopes) > 0 {
		names = slices.Clone(scan.scopes[len(scan.scopes)-1].CommandPath)
	}
	names = append(names, selected.container.Name())
	scan.scopes = append(scan.scopes, codec.CompletionScope{CommandPath: names,
		Flags: make(map[string][]codec.CompletionOccurrence), Arguments: make(map[string][]string)})
	scan.positions = append(scan.positions, nil)
}

func (scan *completionScan) scan(ctx context.Context, root Container) error {
	selected, err := selectContainer(root, nil, scan.profiles)
	if err != nil {
		return err
	}
	scan.addScope(selected)
	end := false
	for i := 0; i < len(scan.request.Words); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		word := scan.request.Words[i]
		active := len(scan.path) - 1
		if !end && i == scan.request.Word && strings.HasPrefix(word, "-") {
			scan.target = codec.CompletionTarget{Kind: codec.CompletionFlag, Scope: active}
			if word == "-" {
				continue
			}
		}
		if !end && word == "--" && i != scan.request.Word {
			end = true
			continue
		}
		if !end && strings.HasPrefix(word, "--") {
			name, inline, hasInline := strings.Cut(word[2:], "=")
			owner, flag := findInputFlag(scan.path, name, 0)
			if flag != nil {
				var values []string
				start := len(word)
				if hasInline {
					values, start = []string{inline}, len(name)+3
				}
				scan.flag(&i, owner, flag, values, start, hasInline)
			}
			continue
		}
		if !end && strings.HasPrefix(word, "-") && word != "-" {
			scan.shorts(&i)
			continue
		}
		if !end && len(scan.scopes[active].Positionals) == 0 && i != scan.request.Word {
			if command, ok := scan.path[active].commandByName[word]; ok {
				selected, err := selectContainer(command, scan.path, scan.profiles)
				if err != nil {
					return err
				}
				scan.addScope(selected)
				continue
			}
		}
		if i == scan.request.Word {
			scan.commandSite = !end && len(scan.scopes[active].Positionals) == 0 && len(scan.path[active].commands) > 0
			scan.target = codec.CompletionTarget{Kind: codec.CompletionArgument, Scope: active}
		}
		scan.scopes[active].Positionals = append(scan.scopes[active].Positionals, word)
		scan.positions[active] = append(scan.positions[active], i)
	}
	// The final path determines inherited codec overrides, just as in parsing.
	plan := scan.path[len(scan.path)-1].container.base().plan
	for i := range scan.path {
		schema, err := selectSchema(&plan.schemas[i], scan.profiles)
		if err != nil {
			return err
		}
		scan.path[i].schema = schema
		scan.assignArguments(i)
	}
	if scan.commandSite && scan.target.Name == "" {
		scan.target.Kind = codec.CompletionCommand
	}
	return ctx.Err()
}

func (scan *completionScan) shorts(index *int) {
	word := scan.request.Words[*index]
	for start := 1; start < len(word); {
		short, size := utf8.DecodeRuneInString(word[start:])
		start += size
		owner, flag := findInputFlag(scan.path, "", short)
		if flag == nil {
			return
		}
		rest := word[start:]
		hasInline := strings.HasPrefix(rest, "=")
		consumes := hasInline || flag.arity.min > 0 || flag.arity.max > 1 || flag.arity.max == -1
		var values []string
		if consumes && rest != "" {
			if hasInline {
				start++
			}
			values = []string{word[start:]}
		}
		scan.flag(index, owner, flag, values, start, consumes && rest != "")
		if consumes {
			return
		}
	}
}

func (scan *completionScan) flag(index *int, owner int, flag *Flag[any, any], values []string, start int, inline bool) {
	word := *index
	occurrence := len(scan.scopes[owner].Flags[flag.Name()])
	if word == scan.request.Word && inline && scan.request.Offset >= start {
		scan.valueTarget(owner, flag, occurrence, 0, scan.request.Words[word][:start])
	}
	if flag.arity.max != 0 && flag.arity != (valueArity{min: 0, max: 1}) {
		active := len(scan.path) - 1
		for *index+1 < len(scan.request.Words) && (flag.arity.max == -1 || len(values) < flag.arity.max) {
			next := scan.request.Words[*index+1]
			if next == "--" {
				break
			}
			// Recover at a known option instead of swallowing it as a missing
			// required value. Literal option-looking values remain expressible inline.
			if len(values) < flag.arity.min && scan.optionBoundary(next, *index+1) {
				break
			}
			if len(values) >= flag.arity.min {
				if strings.HasPrefix(next, "-") && next != "-" {
					break
				}
				if len(scan.scopes[active].Positionals) == 0 {
					if _, command := scan.path[active].commandByName[next]; command && *index+1 != scan.request.Word {
						break
					}
				}
			}
			*index++
			if *index == scan.request.Word {
				scan.valueTarget(owner, flag, occurrence, len(values), "")
			}
			values = append(values, next)
		}
	}
	scan.scopes[owner].Flags[flag.Name()] = append(scan.scopes[owner].Flags[flag.Name()], codec.CompletionOccurrence{
		Values: slices.Clone(values), Missing: len(values) < flag.arity.min, Word: word,
	})
}

func (scan *completionScan) optionBoundary(word string, index int) bool {
	if strings.HasPrefix(word, "--") {
		name, _, _ := strings.Cut(word[2:], "=")
		_, flag := findInputFlag(scan.path, name, 0)
		return flag != nil || index == scan.request.Word
	}
	if strings.HasPrefix(word, "-") && len(word) > 1 {
		short, _ := utf8.DecodeRuneInString(word[1:])
		_, flag := findInputFlag(scan.path, "", short)
		return flag != nil
	}
	return false
}

func (scan *completionScan) valueTarget(owner int, flag *Flag[any, any], occurrence, value int, prefix string) {
	scan.target = codec.CompletionTarget{Kind: codec.CompletionFlagValue, Name: flag.Name(), Scope: owner, Occurrence: occurrence, ValueIndex: value}
	scan.valuePrefix = prefix
}

func (scan *completionScan) assignArguments(scope int) {
	position := 0
	for _, argument := range scan.path[scope].schema.arguments {
		count := len(scan.scopes[scope].Positionals) - position
		if argument.arity.max >= 0 {
			count = min(count, argument.arity.max)
		}
		if count == 0 {
			continue
		}
		scan.scopes[scope].Arguments[argument.Name()] = slices.Clone(scan.scopes[scope].Positionals[position : position+count])
		for i := 0; i < count; i++ {
			if scan.positions[scope][position+i] == scan.request.Word {
				scan.target = codec.CompletionTarget{Kind: codec.CompletionArgument, Name: argument.Name(), Scope: scope, ValueIndex: i}
			}
		}
		position += count
	}
}

func (scan *completionScan) input() codec.CompletionInput {
	word := scan.request.Words[scan.request.Word]
	leaf := scan.scopes[len(scan.scopes)-1]
	input := codec.CompletionInput{
		Words: scan.request.Words, Word: scan.request.Word, Prefix: strings.TrimPrefix(word[:scan.request.Offset], scan.valuePrefix),
		Suffix: word[scan.request.Offset:], Target: scan.target, ValueIndex: scan.target.ValueIndex, Profiles: slices.Clone(scan.profiles),
		CommandPath: leaf.CommandPath, Scopes: scan.scopes, Flags: make(map[string][]codec.CompletionOccurrence),
		Arguments: leaf.Arguments, Positionals: leaf.Positionals,
	}
	// Include only the deepest visible definition for each canonical name.
	for i := range scan.path {
		for _, flag := range scan.path[i].schema.flags {
			delete(input.Flags, flag.Name())
			if occurrences, ok := scan.scopes[i].Flags[flag.Name()]; ok {
				input.Flags[flag.Name()] = occurrences
			}
		}
	}
	if scan.target.Kind == codec.CompletionFlagValue {
		input.Values = scan.scopes[scan.target.Scope].Flags[scan.target.Name][scan.target.Occurrence].Values
	} else if scan.target.Kind == codec.CompletionArgument {
		input.Values = scan.scopes[scan.target.Scope].Arguments[scan.target.Name]
	}
	return input
}

func (scan *completionScan) targetCodec() codec.Codec[any, any] {
	schema := scan.path[scan.target.Scope].schema
	if scan.target.Kind == codec.CompletionFlagValue {
		if flag := schema.flagByName[scan.target.Name]; flag != nil {
			return flag.codec
		}
	}
	if scan.target.Kind == codec.CompletionArgument {
		for _, argument := range schema.arguments {
			if argument.Name() == scan.target.Name {
				return argument.codec
			}
		}
	}
	return nil
}

func (scan *completionScan) targetType() reflect.Type {
	schema := scan.path[scan.target.Scope].schema
	if scan.target.Kind == codec.CompletionFlagValue {
		return schema.flagByName[scan.target.Name].ValueType()
	}
	for _, argument := range schema.arguments {
		if argument.Name() == scan.target.Name {
			return argument.ValueType()
		}
	}
	return nil
}

func (scan *completionScan) flagCandidates() []codec.CompletionCandidate {
	var result []codec.CompletionCandidate
	for i, selected := range scan.path {
		for _, flag := range selected.schema.flags {
			if !flag.IsVisible() {
				continue
			}
			used := false
			for _, occurrence := range scan.scopes[i].Flags[flag.Name()] {
				used = used || occurrence.Word != scan.request.Word
			}
			if used && !flag.IsCumulative() {
				continue
			}
			for _, name := range flag.Names() {
				result = append(result, codec.CompletionCandidate{Value: "--" + name, Description: flag.Description()})
			}
			for _, short := range flag.Shorts() {
				result = append(result, codec.CompletionCandidate{Value: "-" + string(short), Description: flag.Description()})
			}
		}
	}
	return result
}
