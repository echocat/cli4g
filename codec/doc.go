// Package codec parses CLI inputs into typed values and provides codecs for
// custom types, dates, durations, slices, maps, and invocation-owned files.
//
// # Standard values
//
// The CLI package selects standard codecs for bool, string, numeric types,
// time.Duration, time.Time, their slices, and string-keyed maps of these types.
// Dates use [DateTimeCodec] by default;
// select [NewDate] for date-only inputs. Named registrations override automatic
// selection. See [NewDuration], [NewDateTime], [NewSlice], [NewMap], and [NewFile].
// [Defaults] is an overridable factory whose initial value supplies one
// [CombinedDefaults]. Its hard-coded type dispatch binds an existing concrete
// codec during Create; parsing does not repeat registry lookup or type dispatch.
// It also resolves *os.File and []*os.File. Defined types do not match merely by
// their underlying kind, preserving custom Value and text-adapter behavior.
// Configure Defaults before Create without concurrent modification; nil disables
// fallback codecs. Already compiled applications retain their selected codecs.
// Custom registry codecs can implement [TypeBinder] for immutable specialization.
//
// # Maps
//
// [NewMap] supports [MapKey] keys: strings, booleans, integers, and named types
// based on them. Compose the value codec independently, for example
// NewMap[uint32](NewString()) for map[uint32]string. Register non-string-keyed
// maps explicitly or supply their codec directly to a flag or argument builder.
// Shortcuts such as [NewInt32Map] always use string keys.
//
// Each input is key=value, split at the first '='. A map flag consumes one pair
// per occurrence; a map argument consumes remaining pairs and must be last.
// Duplicate decoded keys are validation errors, even with different spellings
// such as 42 and 0x2a. Maps preserve nil/empty defaults and format in stable key
// order. CLI input replaces defaults rather than merging with them.
//
// # Custom values
//
// CLI struct tags automatically adapt concrete T or *T fields whose pointer
// implements [Value], or both encoding.TextMarshaler and encoding.TextUnmarshaler.
// Explicit/registered codecs take precedence; Value wins when both adapters
// apply. Automatic targets must support Set/UnmarshalText on their zero value.
// See [AutomaticFlagValue] and [AutomaticMarshaller].
//
// For explicit adapters, use [NewValue] or [NewValueP] for Set/String types and
// [NewMarshaller] or [NewMarshallerP] for text-marshaled types. Factories must create fresh
// targets and support concurrent calls. Mark user mistakes with [NewValidationError]
// and return technical errors normally. Unmarked errors receive neutral messages
// without losing their causes or operational classification.
//
//	return NewValidationErrorf("port must be between 1 and 65535")
//
// # Error messages and causes
//
// Built-in codecs describe requirements and bounds without echoing the supplied
// value. [NewValidationError] separates the displayed message from diagnostic causes:
//
//	return NewValidationError("an integer expected", parseError)
//
// Error returns only the message; Unwrap returns the causes as []error. errors.Is
// and errors.As can still identify original strconv.NumError, time.ParseError,
// filesystem failures, and other diagnostic causes. [NewMessageError] provides
// the same separation for errors without a validation classification.
//
// Warning: explicitly marked ValidationError messages are displayed verbatim.
// Authors must never include supplied values, passwords, or other confidential
// data. [NewValidationErrorf] with %w formats cause text into its message; use
// NewValidationError(message, causes...) to retain causes without printing them.
// [SanitizeError] trusts marked messages and masks unmarked error text. A caller
// deliberately traversing the cause tree can still access the original input.
//
// # Enums
//
// [Enum] is an optional codec contract enumerating all valid resource-free
// requirements as [EnumVariant] values. Custom implementations remain responsible
// for enforcing their advertised set. Metadata must not resolve runtime values.
// [Raw] preserves typed enums through an Enum[any] adapter.
//
// [NewEnum] restricts an existing codec's requirements; [NewEnumVariants] also
// accepts descriptions. The source codec's parsing/normalization rules remain in
// effect. Enums select one alternative and are not cumulative: wrap them with
// [NewSlice] or [NewMap] for repeated elements. Collections expose their element
// choices through [EnumChoices], without claiming a finite collection space.
// [DisplayEnum] supplies independent, display-safe choices for help and tooling.
//
// [NewMarshallerEnum] and [NewValueEnum] take typed runtime alternatives and retain
// only their canonical text/tokens as requirements. Their Variants counterparts
// add descriptions; their PEnum counterparts use explicit pointer factories.
// Text is derived once during construction, not during help or parsing. Value
// enums require String output to be accepted by Set. Undeclared input tokens are
// rejected before UnmarshalText/Set; Resolve still creates fresh runtime targets.
//
// # Context streams
//
// [ContextWithStdio] supplies borrowed stdin, stdout and stderr streams. Read them
// with [StdioOfContext], [StdinOfContext], [StdoutOfContext] or [StderrOfContext]. These getters
// never fall back to process stdio. CLI applications prepare their operation
// contexts according to their fallback policy; standalone callers may explicitly
// opt in with [ContextWithFallbackToStdio]. Explicit streams, including nil
// stdin/stdout, always take precedence over fallback. Nil stderr uses stdout;
// pass io.Discard to suppress stderr explicitly. Streams are never closed.
// [Print], [Printf] and [Println] write to context stdout; their Err* counterparts
// use context stderr. Missing output is a no-op.
// [FileCodec.WithAllowStdio] requires context streams of type *os.File.
//
// # Requirements and resources
//
// [Codec] uses V for the runtime value and R for input configuration and defaults.
// They may differ: [FileCodec] uses *os.File and [FileRequirement]; value adapters
// use the original input tokens. Requirements must contain no runtime resources.
//
//   - ParseRequirement reads input into R without acquiring resources.
//   - FormatRequirement serializes R losslessly; DisplayRequirement supplies help text.
//   - Resolve creates V and registers acquired resources with [Scope.OnClose].
//
// Register cleanup immediately, including when resolution later fails:
//
//	file, err := os.Open(requirement.Path)
//	if file != nil {
//	    scope.OnClose(file.Close)
//	}
//	return file, err
//
// Do not close codec-owned values again in options cleanup. Codecs are immutable
// and concurrent-safe; operations must honor their context. See [Codec] for the
// serialization contract and [Cumulative] for repeatable flags.
package codec
