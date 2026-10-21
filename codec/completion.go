package codec

import (
	"context"
	"reflect"
	"slices"
	"strings"
)

// Completer optionally supplies context-dependent input suggestions for a codec.
// Unlike Enum, its suggestions need not be exhaustive, stable, or independent of
// other inputs. Implementations may read configuration or query external systems.
// They must honor ctx, close resources acquired for the request, be safe for
// concurrent calls, and leave input unchanged. Completion never calls Resolve or
// application initialization. Keep expensive work out of codec constructors and metadata
// methods: Complete is called only when this codec's value is being completed.
//
// Return Handled=true even for an authoritative empty answer. Handled=false asks
// the framework to try enum metadata and then its ordinary fallback. Errors are
// returned to API callers; shell adapters suppress them to keep editing usable.
// Custom codecs implementing ToRaw must use Raw(ctx, wrapper), so this capability
// survives type erasure. See Complete for enum and collection behavior.
type Completer interface {
	Complete(ctx context.Context, input CompletionInput) (CompletionResult, error)
}

// CompletionFunc adapts a function to Completer.
type CompletionFunc func(context.Context, CompletionInput) (CompletionResult, error)

// Complete invokes f.
func (f CompletionFunc) Complete(ctx context.Context, input CompletionInput) (CompletionResult, error) {
	return f(ctx, input)
}

// CompletionKind identifies the syntactic site being completed.
type CompletionKind string

const (
	CompletionCommand   CompletionKind = "command"    // A command name or positional input.
	CompletionFlag      CompletionKind = "flag"       // A flag name, not its value.
	CompletionFlagValue CompletionKind = "flag-value" // A value belonging to a flag.
	CompletionArgument  CompletionKind = "argument"   // A positional argument's value.
)

// CompletionTarget identifies an input independently of aliases used on the
// command line. Scope indexes CompletionInput.Scopes. Occurrence is zero-based
// for repeated flags; ValueIndex is zero-based within that occurrence or named
// argument. Name is canonical and omits leading dashes.
type CompletionTarget struct {
	Kind       CompletionKind // Syntactic completion site.
	Name       string         // Canonical flag/argument name, empty for command/unknown input.
	Scope      int            // Owner's index in CompletionInput.Scopes.
	Occurrence int
	ValueIndex int
}

// CompletionOccurrence preserves one explicit flag occurrence. Values are
// shell-tokenized but not codec-decoded. Missing means the occurrence has fewer
// than its required number of tokens. A present bare bool has no Values and is
// not Missing; --name="" has one empty token. Word indexes CompletionInput.Words.
type CompletionOccurrence struct {
	Values  []string
	Missing bool
	Word    int
}

// CompletionScope is the input snapshot of one selected container. CommandPath
// includes the application and canonical command names. Maps use canonical names
// and contain explicit inputs only. Positionals preserves all positional tokens,
// including excess input; Arguments groups tokens assigned to known definitions.
type CompletionScope struct {
	CommandPath []string
	Flags       map[string][]CompletionOccurrence
	Arguments   map[string][]string
	Positionals []string
}

// CompletionSource identifies the provenance of an effective input.
type CompletionSource string

const (
	CompletionCLI         CompletionSource = "cli"
	CompletionEnvironment CompletionSource = "environment"
	CompletionDefault     CompletionSource = "default"
)

// CompletionValue contains effective, resource-free input tokens. CLI and
// environment tokens use their ordinary input usage; default tokens are encoded
// for UsageDefault, including structured typed defaults. Environment names the
// selected environment variable, if any. No runtime value is resolved.
type CompletionValue struct {
	Source      CompletionSource // CLI, environment, or declared default.
	Values      []string         // Unresolved tokens in source order.
	Environment string           // Selected environment variable, otherwise empty.
}

// CompletionValues lazily reads effective dependencies of a completion. Flag
// follows CLI > environment > declared default, accepting canonical names and
// long aliases without leading dashes. Argument follows CLI > declared default
// in the deepest selected container.
// An explicit but incomplete CLI occurrence still wins. Factory and runtime
// values are unavailable because completion does not construct options. Returned
// tokens are independent copies. Lookup errors affect only the requested value.
type CompletionValues interface {
	Flag(context.Context, string) (CompletionValue, bool, error)
	Argument(context.Context, string) (CompletionValue, bool, error)
}

// CompletionInput is an invocation-local, best-effort input snapshot. Other
// inputs may be missing, invalid, repeated, or positioned after the cursor.
// Words excludes the executable, includes the current word (possibly empty), and
// preserves otherwise unassigned tokens. Word identifies that word. Prefix and
// Suffix are the codec's portion before/after the cursor, without --flag= or a
// short-flag prefix. Values holds codec-local tokens, including the incomplete
// token at ValueIndex. Collection adapters rebase Values/ValueIndex for their
// element codec while preserving Target and all original scope data. CommandPath
// includes the application; Profiles is sorted.
//
// Flags is the visible canonical-name projection of Scopes. Arguments and
// Positionals refer to the deepest scope. Effective is optional for standalone
// codec callers and provides lazy environment/default access. Snapshot data is
// read-only for the callback and never stored on a shared codec instance.
type CompletionInput struct {
	Words       []string                          // All supplied, shell-tokenized words without the executable.
	Word        int                               // Current word's index in Words.
	Prefix      string                            // Codec-local text before the cursor.
	Suffix      string                            // Codec-local text after the cursor.
	Values      []string                          // Codec-local value tokens, including the current token.
	ValueIndex  int                               // Current index in Values; may differ from Target.ValueIndex for elements.
	Target      CompletionTarget                  // Canonical identity and value index.
	CommandPath []string                          // Application followed by canonical commands.
	Profiles    []string                          // Active, sorted and deduplicated schema profiles.
	Scopes      []CompletionScope                 // Application scope first, selected leaf last.
	Flags       map[string][]CompletionOccurrence // Visible, explicitly supplied flags.
	Arguments   map[string][]string               // Explicit named arguments in the leaf scope.
	Positionals []string                          // Leaf positional tokens in input order, including excess input.
	Effective   CompletionValues                  // Lazy dependency lookup; nil in standalone calls if omitted.
}

// CompletionCandidate is one unquoted input token and optional menu description.
// Value is inserted literally: do not shell-escape it. Description is display
// text only. Adapters handle shell quoting; the CLI core prefix-filters values.
type CompletionCandidate struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// CompletionResult describes suggestions and shell fallback behavior. The zero
// value is unhandled. Handled=true with no candidates suppresses fallbacks.
// Files requests native path completion; Directories restricts it to directories.
// FilePrefix is prepended to path suggestions (for example "key="). NoSpace
// suppresses the usual trailing space; KeepOrder preserves provider order.
// FileInput is filled by the CLI core with the unquoted path search prefix.
// Candidates and file fallback may be combined. Results contain data only.
type CompletionResult struct {
	Candidates  []CompletionCandidate `json:"candidates"`
	Handled     bool                  `json:"handled"`
	Files       bool                  `json:"files,omitempty"`
	Directories bool                  `json:"directories,omitempty"`
	FilePrefix  string                `json:"filePrefix,omitempty"`
	FileInput   string                `json:"fileInput,omitempty"`
	NoSpace     bool                  `json:"noSpace,omitempty"`
	KeepOrder   bool                  `json:"keepOrder,omitempty"`
}

// Complete asks a codec for suggestions without parsing or resolving other
// inputs. An authoritative Completer answer wins. Otherwise actual Enum variants
// are formatted as input tokens, never as potentially redacted display text.
// Multi-token enums match preceding Values and complete ValueIndex.
// Raw, NewSlice, and NewMap preserve this operation; maps delegate only the value
// after the first '='. No EnumChoices display projection is inserted as input.
func Complete[V, R any](ctx context.Context, source Codec[V, R], input CompletionInput) (CompletionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CompletionResult{}, err
	}
	if provider, ok := source.(Completer); ok {
		result, err := provider.Complete(ctx, input)
		if err != nil {
			return CompletionResult{}, SanitizeError(err, "completion could not be generated")
		}
		if err := ctx.Err(); err != nil {
			return CompletionResult{}, err
		}
		if result.Handled {
			result.Candidates = slices.Clone(result.Candidates)
			return result, nil
		}
	}
	enumeration, ok := source.(Enum[R])
	if !ok {
		if reflect.TypeFor[V]().Kind() == reflect.String {
			return CompletionResult{Handled: true, Files: true}, nil
		}
		return CompletionResult{}, nil
	}
	variants, err := enumeration.EnumVariants(ctx)
	if err != nil {
		return CompletionResult{}, SanitizeError(err, "enum completion could not be read")
	}
	result := CompletionResult{Handled: true, KeepOrder: true}
	index := input.ValueIndex
	for _, variant := range variants {
		if err := ctx.Err(); err != nil {
			return CompletionResult{}, err
		}
		values, err := source.FormatRequirement(ctx, variant.Value)
		if err != nil {
			return CompletionResult{}, SanitizeError(err, "enum completion could not be formatted")
		}
		if index < 0 || index >= len(values) || len(input.Values) < index || !slices.Equal(input.Values[:index], values[:index]) {
			continue
		}
		result.Candidates = append(result.Candidates, CompletionCandidate{Value: values[index], Description: variant.Description})
	}
	return result, ctx.Err()
}

func (this raw[V, R]) Complete(ctx context.Context, input CompletionInput) (CompletionResult, error) {
	return Complete(ctx, this.source, input)
}

func (this slice[V, R]) Complete(ctx context.Context, input CompletionInput) (CompletionResult, error) {
	input.ValueIndex = 0
	input.Values = []string{input.Prefix + input.Suffix}
	return Complete(ctx, this.element, input)
}

func (this mapCodec[K, V, R]) Complete(ctx context.Context, input CompletionInput) (CompletionResult, error) {
	key, value, found := strings.Cut(input.Prefix, "=")
	if !found {
		return CompletionResult{Handled: true}, ctx.Err()
	}
	input.Prefix = value
	input.ValueIndex = 0
	input.Values = []string{value + input.Suffix}
	result, err := Complete(ctx, this.element, input)
	if err != nil {
		return CompletionResult{}, err
	}
	if !result.Handled {
		result.Handled = true
	}
	for i := range result.Candidates {
		result.Candidates[i].Value = key + "=" + result.Candidates[i].Value
	}
	result.FilePrefix = key + "=" + result.FilePrefix
	return result, nil
}

func (boolean) Complete(ctx context.Context, _ CompletionInput) (CompletionResult, error) {
	return CompletionResult{Handled: true, Candidates: []CompletionCandidate{{Value: "true"}, {Value: "false"}}}, ctx.Err()
}

// Complete requests native path completion without opening any files.
func (FileCodec) Complete(ctx context.Context, _ CompletionInput) (CompletionResult, error) {
	return CompletionResult{Handled: true, Files: true}, ctx.Err()
}
