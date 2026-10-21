package cli

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/echocat/cli4g/codec"
)

// CompletionRequest identifies a cursor in shell-tokenized input, excluding the
// executable. Word is a zero-based index into Words; Offset is a byte offset in
// that word, not a terminal column or UTF-16 offset. Word=len(Words), Offset=0
// inserts an empty current word at the end. Words may include input after the
// cursor. Shell quoting has already been removed, but codecs have not run.
// Invalid cursor coordinates are errors; incomplete CLI syntax is not.
type CompletionRequest struct {
	Words  []string `json:"words"`
	Word   int      `json:"word"`
	Offset int      `json:"offset"`
}

// Complete provides context-aware suggestions without creating, parsing,
// validating, initializing, or executing application options. It uses the active
// schema selected by opts, including inherited flags and path-specific codecs.
// Unknown or incomplete input is retained best-effort, not reported as an input
// error. Schema/profile errors, invalid coordinates and provider errors return
// directly, without invoking the application's error handler.
//
// Codecs may implement codec.Completer. They receive explicit input maps, named
// and ordered arguments, command scopes, the target, and lazy environment/default
// access through codec.CompletionInput.Effective. Enum, bool, file and collection
// codecs work automatically. Name completion never invokes value completers.
// An explicit flag/argument CompletionHandler takes precedence over its codec;
// Handled=false delegates to the next completion source.
// Suggestions are case-sensitive prefix matches, deduplicated and sorted unless
// KeepOrder is requested. Candidates are whole, unquoted replacement words.
// The caller owns ctx's deadline; provider callbacks must honor cancellation.
// Calls are independent and concurrent-safe when user callbacks are safe.
func (this *Application[O]) Complete(ctx context.Context, opts *ParseOpts, request CompletionRequest) (codec.CompletionResult, error) {
	ctx, err := this.prepareParseContext(ctx)
	if err != nil {
		return codec.CompletionResult{}, err
	}
	request.Words = slices.Clone(request.Words)
	if request.Word == len(request.Words) && request.Offset == 0 {
		request.Words = append(request.Words, "")
	}
	if request.Word < 0 || request.Word >= len(request.Words) || request.Offset < 0 || request.Offset > len(request.Words[request.Word]) {
		return codec.CompletionResult{}, fmt.Errorf("invalid completion cursor")
	}
	var profiles selectedProfiles
	if opts != nil {
		profiles = slices.Clone(opts.Profiles)
		slices.Sort(profiles)
		profiles = slices.Compact(profiles)
	}
	for _, profile := range profiles {
		if !this.ContainsProfile(profile) {
			return codec.CompletionResult{}, NewValidationErrorf("unknown completion profile %q", profile)
		}
	}
	ctx = withProfilesContext(ctx, profiles)
	scan := completionScan{request: request, profiles: profiles}
	if err := scan.scan(ctx, this); err != nil {
		return codec.CompletionResult{}, err
	}
	input := scan.input()
	input.Effective = &completionValues{scan: &scan, opts: opts}
	ctx = withContainerContext(ctx, scan.path[len(scan.path)-1].container)
	result := codec.CompletionResult{Handled: true}
	if input.Target.Kind == codec.CompletionFlag {
		result.Candidates = scan.flagCandidates()
	} else {
		if scan.commandSite {
			for _, command := range scan.path[input.Target.Scope].commands {
				if command.IsVisible() {
					for _, name := range command.Names() {
						result.Candidates = append(result.Candidates, codec.CompletionCandidate{Value: name, Description: command.Description()})
					}
				}
			}
		}
		if source := scan.targetCodec(); source != nil {
			values, err := codec.Complete(ctx, source, input)
			if err != nil {
				return codec.CompletionResult{}, err
			}
			if !values.Handled {
				values.Handled = true
				values.Files = scan.targetType() != nil && scan.targetType().Kind() == reflect.String
			}
			values.Candidates = append(result.Candidates, values.Candidates...)
			result = values
		} else if !scan.commandSite {
			result.Files = true
		}
	}
	if err := ctx.Err(); err != nil {
		return codec.CompletionResult{}, err
	}
	return finishCompletion(result, input.Prefix, scan.valuePrefix), nil
}

func finishCompletion(result codec.CompletionResult, prefix, valuePrefix string) codec.CompletionResult {
	seen := make(map[string]bool)
	filtered := make([]codec.CompletionCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		if strings.HasPrefix(candidate.Value, prefix) && !seen[candidate.Value] {
			seen[candidate.Value] = true
			candidate.Value = valuePrefix + candidate.Value
			filtered = append(filtered, candidate)
		}
	}
	if !result.KeepOrder {
		slices.SortStableFunc(filtered, func(a, b codec.CompletionCandidate) int { return strings.Compare(a.Value, b.Value) })
	}
	result.Candidates = filtered
	if result.Files || result.Directories {
		result.FileInput = strings.TrimPrefix(prefix, result.FilePrefix)
	}
	result.FilePrefix = valuePrefix + result.FilePrefix
	return result
}
