// Package codec parses CLI inputs into typed values and provides codecs for
// custom types, dates, durations, slices, and invocation-owned files.
//
// # Standard values
//
// The CLI package selects standard codecs for bool, string, numeric types,
// time.Duration, time.Time, and their slices. Dates use [DateTimeCodec] by default;
// select [NewDate] for date-only inputs. Named registrations override automatic
// selection. See [NewDuration], [NewDateTime], [NewSlice], and [NewFile].
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
