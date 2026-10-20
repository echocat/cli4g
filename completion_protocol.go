package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/echocat/cli4g/codec"
)

// CompletionProtocolVersion identifies the stable adapter protocol. Its decimal
// representation follows the configured hidden request command (by default
// "__complete"). Version 1 has these transports:
//
//   - __complete 1 bash POINT LINE CURRENT: POINT is a byte offset into the Bash input
//     LINE. CURRENT is Bash's current word after COMP_WORDBREAKS. Simple shell
//     quotes/escapes and command separators are recognized without evaluating
//     expansions, substitutions or shell code. Output is NUL-delimited: "1", path
//     mode (0 none, 1 files, 2 directories), NoSpace (0/1), KeepOrder (0/1), path
//     insertion prefix, path search prefix, Readline replacement prefix to remove
//     from composed path suggestions, then candidate values. All strings
//     are unquoted; the adapter handles Readline word breaks and native paths.
//   - __complete 1 nu: stdin contains exactly one JSON CompletionRequest, limited to
//     1 MiB. An optional raw=true additionally removes simple Nu literal quoting
//     from span tokens; it never evaluates expressions. stdout is a JSON
//     codec.CompletionResult. Input excludes the executable.
//   - __complete 1 powershell: the same JSON request/result as nu, without raw
//     token normalization, each wrapped in standard Base64. The decoded request
//     is limited to 1 MiB. ASCII-only transport preserves Unicode even with
//     Windows PowerShell 5.1's legacy native-pipeline encoding. The adapter sends
//     literal AST values, keeps arguments after the cursor, and converts UTF-16
//     cursor positions to UTF-8 byte offsets without evaluating expressions.
//
// New transports may be added without changing existing transports. Query errors
// become authoritative empty responses with no diagnostics; direct Complete
// callers retain errors. Adapters should invoke the currently selected binary,
// never a path/version captured at installation, and only on completion requests.
const CompletionProtocolVersion = 1

// CompletionQueryTimeout is the cooperative deadline applied to shell queries.
// Direct Complete callers choose their own deadline. Network clients and custom
// completers must honor cancellation; Go cannot forcibly stop an arbitrary user
// callback. Startup and code run before cli4g are outside this budget.
const CompletionQueryTimeout = 500 * time.Millisecond

func (this *Application[O]) queryCompletion(ctx context.Context, opts *ParseOpts, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, CompletionQueryTimeout)
	defer cancel()
	result := codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{}}
	if len(args) < 2 || args[0] != strconv.Itoa(CompletionProtocolVersion) {
		return nil
	}
	output := codec.StdoutOfContext(ctx)
	if output == nil {
		output = io.Discard
	}
	switch CompletionShell(args[1]) {
	case CompletionBash:
		var request CompletionRequest
		var err error
		if len(args) == 5 {
			var point int
			point, err = strconv.Atoi(args[2])
			if err == nil {
				request, err = bashCompletionRequest(args[3], point)
			}
			if err == nil {
				result, err = this.Complete(ctx, opts, request)
			}
		}
		if err != nil {
			result = codec.CompletionResult{Handled: true}
		}
		pathInput := result.FileInput
		replacementPrefix := ""
		if err == nil && len(request.Words) > request.Word && request.Word >= 0 {
			prefix := request.Words[request.Word][:request.Offset]
			if len(args) == 5 && strings.HasSuffix(prefix, args[4]) {
				trim := prefix[:len(prefix)-len(args[4])]
				replacementPrefix = trim
				for i := range result.Candidates {
					result.Candidates[i].Value = strings.TrimPrefix(result.Candidates[i].Value, trim)
				}
			}
		}
		mode := 0
		if result.Files {
			mode = 1
		}
		if result.Directories {
			mode = 2
		}
		if _, err := fmt.Fprintf(output, "1%c%d%c%d%c%d%c%s%c%s%c%s%c", 0, mode, 0, boolInt(result.NoSpace), 0, boolInt(result.KeepOrder), 0, result.FilePrefix, 0, pathInput, 0, replacementPrefix, 0); err != nil {
			return err
		}
		for _, candidate := range result.Candidates {
			if strings.ContainsRune(candidate.Value, 0) {
				continue
			}
			if _, err := fmt.Fprintf(output, "%s%c", candidate.Value, 0); err != nil {
				return err
			}
		}
		return nil
	case CompletionNushell, CompletionPowerShell:
		var request struct {
			CompletionRequest
			Raw bool `json:"raw,omitempty"`
		}
		if input := codec.StdinOfContext(ctx); input != nil {
			if CompletionShell(args[1]) == CompletionPowerShell {
				input = base64.NewDecoder(base64.StdEncoding, input)
			}
			data, readErr := io.ReadAll(io.LimitReader(input, (1<<20)+1))
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err == nil && readErr == nil && len(data) <= 1<<20 {
				var trailing any
				if decoder.Decode(&trailing) == io.EOF {
					if request.Raw && CompletionShell(args[1]) == CompletionNushell {
						unquoteNuCompletion(&request.CompletionRequest)
					}
					if completed, err := this.Complete(ctx, opts, request.CompletionRequest); err == nil {
						result = completed
					}
				}
			}
		}
		if CompletionShell(args[1]) == CompletionPowerShell {
			encoder := base64.NewEncoder(base64.StdEncoding, output)
			if err := json.NewEncoder(encoder).Encode(result); err != nil {
				_ = encoder.Close()
				return err
			}
			return encoder.Close()
		}
		return json.NewEncoder(output).Encode(result)
	}
	return nil
}

func unquoteNuCompletion(request *CompletionRequest) {
	for i, word := range request.Words {
		if i == request.Word && request.Offset >= 0 && request.Offset <= len(word) {
			request.Offset = len(unquoteNuWord(word[:request.Offset]))
		}
		request.Words[i] = unquoteNuWord(word)
	}
}

func unquoteNuWord(word string) string {
	prefix := ""
	if strings.HasPrefix(word, "-") {
		if name, value, found := strings.Cut(word, "="); found {
			prefix, word = name+"=", value
		}
	}
	if len(word) == 0 || (word[0] != '\'' && word[0] != '"' && word[0] != '`') {
		return prefix + word
	}
	quote := word[0]
	if len(word) == 1 || word[len(word)-1] != quote {
		word += string(quote)
	}
	if quote == '"' {
		if value, err := strconv.Unquote(word); err == nil {
			return prefix + value
		}
	}
	return prefix + word[1:len(word)-1]
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
