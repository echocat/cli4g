package codec

import (
	"context"
	"reflect"
	"time"
)

// DateTimeCodec parses time.Time values using RFC3339, RFC3339Nano, DateTime,
// RFC822, RFC822Z, and all DateCodec default layouts. Date-only inputs resolve
// to midnight. The zero value uses UTC for inputs without a zone. RFC822 zone
// abbreviations other than UTC/GMT must be known to the configured location.
// Input offsets and nanoseconds are preserved. Default help uses RFC3339Nano;
// custom codecs display using their first layout. Monotonic readings are not retained.
type DateTimeCodec struct{ calendarConfig }

// NewDateTime creates the standard UTC date/time codec; see [DateTimeCodec].
func NewDateTime() DateTimeCodec { return DateTimeCodec{} }

// WithFormats replaces the input layouts, in matching order. The first layout
// is also used for display. Layouts use Go's reference time, not strftime.
func (this DateTimeCodec) WithFormats(first string, additional ...string) DateTimeCodec {
	this.formats = calendarFormats(first, additional)
	return this
}

// WithLocation selects the location for inputs without a zone; nil selects UTC.
// Use the same location for typed defaults that carry an IANA time zone, so
// requirement cloning retains its daylight-saving rules without loading files.
func (this DateTimeCodec) WithLocation(location *time.Location) DateTimeCodec {
	this.location = location
	return this
}

func (this DateTimeCodec) calendar() calendarCodec { return calendarCodec{this.calendarConfig, false} }

func (this DateTimeCodec) ToRaw(ctx context.Context) Codec[any, any] {
	if this.calendar().validate() != nil {
		return nil
	}
	return Raw(ctx, this)
}

func (this DateTimeCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return this.calendar().CanHandleType(ctx, typ)
}

func (DateTimeCodec) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this DateTimeCodec) ParseRequirement(ctx context.Context, usage Usage, args ...string) (time.Time, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return time.Time{}, err
	}
	return this.calendar().parse(ctx, usage, args[0])
}

func (this DateTimeCodec) FormatRequirement(ctx context.Context, value time.Time) ([]string, error) {
	return this.calendar().format(ctx, value)
}

func (this DateTimeCodec) DisplayRequirement(ctx context.Context, value time.Time) ([]string, error) {
	return this.calendar().display(ctx, value)
}

func (this DateTimeCodec) CloneRequirement(ctx context.Context, value time.Time) (time.Time, error) {
	return this.calendar().clone(ctx, value)
}

func (this DateTimeCodec) Resolve(ctx context.Context, _ Scope, value time.Time) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	if err := this.calendar().validate(); err != nil {
		return time.Time{}, err
	}
	return this.calendar().normalize(value), nil
}
