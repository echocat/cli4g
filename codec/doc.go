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
// Adapt Set/String types with [NewValue] or [NewValueP]. Adapt text-marshaled
// types with [NewMarshaller] or [NewMarshallerP]. Factories must create fresh
// targets and support concurrent calls. These adapters preserve returned errors:
// mark user mistakes with [NewValidationError], and return technical errors normally.
//
//	return NewValidationErrorf("port must be between 1 and 65535")
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
