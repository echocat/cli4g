package codec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"
)

type calendarConfig struct {
	formats  []string
	location *time.Location
}

type calendarCodec struct {
	calendarConfig
	dateOnly bool
}

func calendarFormats(first string, additional []string) []string {
	result := make([]string, 1+len(additional))
	result[0] = first
	copy(result[1:], additional)
	return result
}

func (this calendarCodec) layouts() []string {
	if this.formats != nil {
		return this.formats
	}
	dates := []string{time.DateOnly, "02 Jan 06", "02.01.2006", "20060102"}
	if this.dateOnly {
		return dates
	}
	return append([]string{time.RFC3339, time.RFC3339Nano, time.DateTime, time.RFC822, time.RFC822Z}, dates...)
}

func (this calendarCodec) zone() *time.Location {
	if this.location == nil {
		return time.UTC
	}
	return this.location
}

func (this calendarCodec) prefix() string {
	if this.dateOnly {
		return "date+json:"
	}
	return "datetime+json:"
}

func (this calendarCodec) validate() error {
	if slices.Contains(this.layouts(), "") {
		return fmt.Errorf("%w: date/time layouts must not be empty", ErrInvalid)
	}
	return nil
}

func (this calendarCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := this.validate(); err != nil {
		return false, err
	}
	return typ == reflect.TypeFor[time.Time](), nil
}

func (this calendarCodec) normalize(value time.Time) time.Time {
	if this.dateOnly {
		year, month, day := value.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
	}
	return value.Round(0)
}

func (this calendarCodec) parse(ctx context.Context, usage Usage, input string) (time.Time, error) {
	if err := this.validate(); err != nil {
		return time.Time{}, err
	}
	if usage == UsageDefault && strings.HasPrefix(input, this.prefix()) {
		value, err := this.parseWire(input)
		if err != nil {
			return time.Time{}, err
		}
		return value, ctx.Err()
	}
	var last error
	for _, layout := range this.layouts() {
		if err := ctx.Err(); err != nil {
			return time.Time{}, err
		}
		value, err := time.ParseInLocation(layout, input, this.zone())
		if err != nil {
			last = err
			continue
		}
		// time.Parse invents a zero-offset location for unknown abbreviations.
		// Reject that fallback rather than silently changing the requested instant.
		name, offset := value.Zone()
		hasZoneOffset := strings.Contains(layout, "-07") || strings.Contains(layout, "Z07")
		if strings.Contains(layout, "MST") && !hasZoneOffset && offset == 0 && name != "UTC" && name != "GMT" && value.Location() != this.zone() {
			last = fmt.Errorf("time zone abbreviation %q is unknown in %s", name, this.zone())
			continue
		}
		if this.dateOnly {
			hour, minute, second := value.Clock()
			if hour != 0 || minute != 0 || second != 0 || value.Nanosecond() != 0 {
				last = fmt.Errorf("date inputs must not include a nonzero time of day")
				continue
			}
		}
		return this.normalize(value), ctx.Err()
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q using layouts %q: %w", ErrIllegalArgument, input, this.layouts(), last)
}

// Default serialization is independent of display layouts. In particular it
// preserves nanoseconds, fixed zone names/offsets, and configured zone rules.
// Monotonic clock readings are runtime-only and are intentionally removed.
type calendarWire struct {
	Time   []byte `json:"time"`
	Zone   string `json:"zone"`
	Name   []byte `json:"name,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

func (this calendarCodec) requirementZone(value time.Time) (calendarWire, error) {
	wire := calendarWire{}
	switch value.Location() {
	case time.UTC:
		wire.Zone = "utc"
	case this.zone():
		wire.Zone = "configured"
	case time.Local:
		wire.Zone = "local"
	default:
		start, end := value.ZoneBounds()
		name, offset := value.Zone()
		if !start.IsZero() || !end.IsZero() || value.Location().String() != name {
			return wire, fmt.Errorf("%w: requirement location %q must be configured with WithLocation", ErrInvalid, value.Location())
		}
		wire.Zone = "fixed"
		wire.Name, wire.Offset = []byte(name), offset
	}
	return wire, nil
}

func (this calendarCodec) format(ctx context.Context, value time.Time) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := this.validate(); err != nil {
		return nil, err
	}
	value = this.normalize(value)
	wire, err := this.requirementZone(value)
	if err != nil {
		return nil, err
	}
	wire.Time, err = value.UTC().MarshalBinary()
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	return []string{this.prefix() + string(encoded)}, ctx.Err()
}

func (this calendarCodec) parseWire(input string) (time.Time, error) {
	var wire calendarWire
	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(input, this.prefix())))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return time.Time{}, fmt.Errorf("%w: date/time requirement: %w", ErrIllegalArgument, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return time.Time{}, fmt.Errorf("%w: trailing date/time requirement data", ErrIllegalArgument)
	}
	var value time.Time
	if err := value.UnmarshalBinary(wire.Time); err != nil {
		return time.Time{}, fmt.Errorf("%w: date/time requirement: %w", ErrIllegalArgument, err)
	}
	switch wire.Zone {
	case "utc":
		value = value.UTC()
	case "configured":
		value = value.In(this.zone())
	case "local":
		value = value.In(time.Local)
	case "fixed":
		value = value.In(time.FixedZone(string(wire.Name), wire.Offset))
	default:
		return time.Time{}, fmt.Errorf("%w: unknown requirement zone %q", ErrIllegalArgument, wire.Zone)
	}
	return this.normalize(value), nil
}

func (this calendarCodec) display(ctx context.Context, value time.Time) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := this.validate(); err != nil {
		return nil, err
	}
	layout := this.layouts()[0]
	if this.formats == nil && !this.dateOnly {
		layout = time.RFC3339Nano
	}
	return []string{this.normalize(value).Format(layout)}, nil
}

func (this calendarCodec) clone(ctx context.Context, value time.Time) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	if err := this.validate(); err != nil {
		return time.Time{}, err
	}
	value = this.normalize(value)
	if _, err := this.requirementZone(value); err != nil {
		return time.Time{}, err
	}
	// time.Time and its locations are immutable, so no JSON roundtrip is needed.
	return value, ctx.Err()
}
