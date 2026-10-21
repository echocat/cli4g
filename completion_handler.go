package cli

import "github.com/echocat/cli4g/codec"

// CompletionHandler supplies value suggestions using the automatically built
// completion context. It is the same callback accepted by
// codec.WithCompletionHandler; no duplicated CLI options model is required.
//
// Handlers on a flag or argument run before that input's codec completer. Return
// Handled=false to delegate, or Handled=true for an authoritative answer, including
// no candidates. Commands, flag names, prefix filtering and shell quoting remain
// automatic. Callbacks must honor cancellation and be concurrent-safe.
type CompletionHandler = codec.CompletionFunc
