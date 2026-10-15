package codec

import (
	"context"
	"reflect"
	"time"
)

// DateCodec parses dates as time.Time values at midnight. The default layouts
// are time.DateOnly, "02 Jan 06", "02.01.2006", and "20060102". The zero value
// uses these layouts and UTC. Typed requirements are normalized to midnight.
// Default help displays ISO dates. Two-digit years use Go's 69/68 century split.
type DateCodec struct{ calendarConfig }

// NewDate creates a UTC, date-only codec with the layouts listed in [DateCodec].
func NewDate() DateCodec { return DateCodec{} }

// WithFormats replaces the input layouts, in matching order. The first layout
// is also used for display. Layouts use Go's reference time, not strftime.
func (this DateCodec) WithFormats(first string, additional ...string) DateCodec {
	this.formats = calendarFormats(first, additional)
	return this
}

// WithLocation selects the location for inputs without a zone; nil selects UTC.
// Use the same location for typed defaults that carry an IANA time zone.
func (this DateCodec) WithLocation(location *time.Location) DateCodec {
	this.location = location
	return this
}

func (this DateCodec) calendar() calendarCodec { return calendarCodec{this.calendarConfig, true} }

func (this DateCodec) ToRaw(ctx context.Context) Codec[any, any] {
	if this.calendar().validate() != nil {
		return nil
	}
	return Raw(ctx, this)
}

func (this DateCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return this.calendar().CanHandleType(ctx, typ)
}

func (DateCodec) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this DateCodec) ParseRequirement(ctx context.Context, usage Usage, args ...string) (time.Time, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return time.Time{}, err
	}
	return this.calendar().parse(ctx, usage, args[0])
}

func (this DateCodec) FormatRequirement(ctx context.Context, value time.Time) ([]string, error) {
	return this.calendar().format(ctx, value)
}

func (this DateCodec) DisplayRequirement(ctx context.Context, value time.Time) ([]string, error) {
	return this.calendar().display(ctx, value)
}

func (this DateCodec) CloneRequirement(ctx context.Context, value time.Time) (time.Time, error) {
	return this.calendar().clone(ctx, value)
}

func (this DateCodec) Resolve(ctx context.Context, _ Scope, value time.Time) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	if err := this.calendar().validate(); err != nil {
		return time.Time{}, err
	}
	return this.calendar().normalize(value), nil
}
