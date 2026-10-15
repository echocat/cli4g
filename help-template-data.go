package cli

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/echocat/cli4g/codec"
)

// HelpTemplateData supplies visible commands, inputs, and display-safe defaults
// to application, command, and usage templates. Embedded [HelpInfo] also contains
// hidden definitions. DefaultText uses codec DisplayRequirement, not serialization.
type HelpTemplateData struct {
	HelpInfo
	Invocation     InvocationLike
	FullName       string
	NamePath       []string
	Description    string
	Usage          string
	Commands       []HelpCommandTemplateData
	Flags          []HelpFlagTemplateData
	InheritedFlags []HelpFlagTemplateData
	Arguments      []HelpArgumentTemplateData
}

// HelpCommandTemplateData is the input to the command-entry template.
type HelpCommandTemplateData struct {
	Definition  CommandLike
	Names       []string
	FullName    string
	Label       string
	Description string
}

// HelpFlagTemplateData is the input to the flag template, including safe default text.
type HelpFlagTemplateData struct {
	HelpFlagInfo
	Names                 []string
	Shorts                []rune
	Label                 string
	Description           string
	Placeholder           string
	ValueType             reflect.Type
	Required              bool
	EnvVars               []string
	HasDefault            bool
	Default               []string
	DefaultText           string
	Choices               []HelpEnumChoice
	ChoicesText           string
	HasChoiceDescriptions bool
}

// HelpArgumentTemplateData is the input to the argument template.
type HelpArgumentTemplateData struct {
	HelpArgumentInfo
	Name                  string
	Label                 string
	Description           string
	Placeholder           string
	ValueType             reflect.Type
	Required              bool
	HasDefault            bool
	Default               []string
	DefaultText           string
	Choices               []HelpEnumChoice
	ChoicesText           string
	HasChoiceDescriptions bool
}

// HelpEnumChoice describes an accepted scalar or collection-element value.
type HelpEnumChoice struct {
	Values      []string
	Text        string
	Description string
}

func buildHelpTemplateData(ctx context.Context, invocation InvocationLike) (HelpTemplateData, error) {
	if invocation == nil || (reflect.ValueOf(invocation).Kind() == reflect.Pointer && reflect.ValueOf(invocation).IsNil()) {
		return HelpTemplateData{}, fmt.Errorf("help requires an invocation")
	}
	info := invocation.HelpInfo()
	if info.Target == nil || len(info.Path) == 0 {
		return HelpTemplateData{}, fmt.Errorf("help invocation has no selected container")
	}
	data := HelpTemplateData{
		HelpInfo:    info,
		Invocation:  invocation,
		Description: info.Target.Description(),
	}
	for _, container := range info.Path {
		data.NamePath = append(data.NamePath, container.Name())
	}
	data.FullName = strings.Join(data.NamePath, " ")
	data.Usage = data.FullName
	for _, command := range info.Commands {
		if !command.IsVisible() {
			continue
		}
		data.Commands = append(data.Commands, HelpCommandTemplateData{
			Definition:  command,
			Names:       command.Names(),
			Label:       strings.Join(command.Names(), ", "),
			FullName:    data.FullName + " " + command.Name(),
			Description: command.Description(),
		})
	}
	for _, flag := range info.Flags {
		if err := ctx.Err(); err != nil {
			return data, err
		}
		definition := flag.Definition
		if !definition.IsVisible() {
			continue
		}
		formatted, hasDefault, err := definition.FormatDefault(ctx)
		if err != nil {
			return data, fmt.Errorf("help flag --%s default: %w", definition.Name(), err)
		}
		choices, choicesText, described, err := helpEnumChoices(ctx, definition.Codec())
		if err != nil {
			return data, fmt.Errorf("help flag --%s choices: %w", definition.Name(), err)
		}
		entry := HelpFlagTemplateData{
			HelpFlagInfo:          flag,
			Names:                 definition.Names(),
			Shorts:                definition.Shorts(),
			Label:                 helpFlagLabel(flag),
			Description:           definition.Description(),
			Placeholder:           definition.Placeholder(),
			ValueType:             definition.ValueType(),
			Required:              definition.IsRequired(),
			EnvVars:               definition.EnvVars(),
			HasDefault:            hasDefault,
			Default:               formatted,
			DefaultText:           helpDefaultText(formatted),
			Choices:               choices,
			ChoicesText:           choicesText,
			HasChoiceDescriptions: described,
		}
		if flag.Inherited {
			data.InheritedFlags = append(data.InheritedFlags, entry)
		} else {
			data.Flags = append(data.Flags, entry)
		}
	}
	if len(data.Flags)+len(data.InheritedFlags) > 0 {
		data.Usage += " [flags]"
	}
	if len(data.Commands) > 0 {
		data.Usage += " [command]"
	}
	for _, argument := range info.Arguments {
		if err := ctx.Err(); err != nil {
			return data, err
		}
		definition := argument.Definition
		formatted, hasDefault, err := definition.FormatDefault(ctx)
		if err != nil {
			return data, fmt.Errorf("help argument %s default: %w", definition.Name(), err)
		}
		choices, choicesText, described, err := helpEnumChoices(ctx, definition.Codec())
		if err != nil {
			return data, fmt.Errorf("help argument %s choices: %w", definition.Name(), err)
		}
		label := helpValueLabel(definition.Placeholder(), definition.Name(), argument.MinValues, argument.MaxValues)
		usageLabel := label
		if definition.IsRequired() {
			usageLabel = "<" + usageLabel + ">"
		} else {
			usageLabel = "[" + usageLabel + "]"
		}
		if argument.MaxValues != 0 {
			data.Usage += " " + usageLabel
		}
		data.Arguments = append(data.Arguments, HelpArgumentTemplateData{
			HelpArgumentInfo:      argument,
			Name:                  definition.Name(),
			Label:                 label,
			Description:           definition.Description(),
			Placeholder:           definition.Placeholder(),
			ValueType:             definition.ValueType(),
			Required:              definition.IsRequired(),
			HasDefault:            hasDefault,
			Default:               formatted,
			DefaultText:           helpDefaultText(formatted),
			Choices:               choices,
			ChoicesText:           choicesText,
			HasChoiceDescriptions: described,
		})
	}
	return data, ctx.Err()
}

func helpEnumChoices(ctx context.Context, source codec.Codec[any, any]) ([]HelpEnumChoice, string, bool, error) {
	variants, err := codec.DisplayEnum(ctx, source)
	if err != nil {
		return nil, "", false, err
	}
	var choices []HelpEnumChoice
	var labels []string
	described := false
	for _, variant := range variants {
		label := helpDefaultText(variant.Values)
		choices = append(choices, HelpEnumChoice{Values: variant.Values, Text: label, Description: variant.Description})
		labels = append(labels, label)
		described = described || variant.Description != ""
	}
	return choices, strings.Join(labels, ", "), described, ctx.Err()
}

func helpFlagLabel(info HelpFlagInfo) string {
	flag := info.Definition
	var names []string
	for _, short := range flag.Shorts() {
		names = append(names, "-"+string(short))
	}
	for _, name := range flag.Names() {
		names = append(names, "--"+name)
	}
	label := strings.Join(names, ", ")
	if info.MaxValues == 0 || flag.IsBoolFlag() {
		return label
	}
	value := helpValueLabel(flag.Placeholder(), flag.Name(), info.MinValues, info.MaxValues)
	if info.MinValues == 0 && info.MaxValues == 1 {
		return label + "[=" + value + "]"
	}
	return label + " " + value
}

func helpValueLabel(placeholder, name string, minValues, maxValues int) string {
	if placeholder == "" {
		placeholder = strings.ToUpper(name)
	}
	if maxValues == -1 {
		return placeholder + "..."
	}
	if maxValues > 1 {
		return fmt.Sprintf("%s{%d..%d}", placeholder, minValues, maxValues)
	}
	return placeholder
}

func helpDefaultText(values []string) string {
	if values == nil {
		return "nil"
	}
	if len(values) == 0 {
		return "[]"
	}
	formatted := make([]string, len(values))
	for i, value := range values {
		if value == "" || strings.ContainsAny(value, " \t\r\n,\"\\") {
			value = strconv.Quote(value)
		}
		formatted[i] = value
	}
	if len(formatted) == 1 {
		return formatted[0]
	}
	return "[" + strings.Join(formatted, ", ") + "]"
}
