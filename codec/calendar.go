package codec

import (
	"context"
	"encoding/json"
	"errors"
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
	causes := []error{ErrIllegalArgument}
	var reason string
	for _, layout := range this.layouts() {
		if err := ctx.Err(); err != nil {
			return time.Time{}, err
		}
		value, err := time.ParseInLocation(layout, input, this.zone())
		if err != nil {
			causes = append(causes, err)
			var detail *time.ParseError
			if errors.As(err, &detail) {
				switch detail.Message {
				case ": month out of range":
					reason = "month should be between 1 and 12"
				case ": day out of range":
					reason = "day must exist in the selected month"
				case ": hour out of range":
					reason = "hour should be between 0 and 23"
				case ": minute out of range":
					reason = "minute should be between 0 and 59"
				case ": second out of range":
					reason = "second should be between 0 and 59"
				}
			}
			continue
		}
		// time.Parse invents a zero-offset location for unknown abbreviations.
		// Reject that fallback rather than silently changing the requested instant.
		name, offset := value.Zone()
		hasZoneOffset := strings.Contains(layout, "-07") || strings.Contains(layout, "Z07")
		if strings.Contains(layout, "MST") && !hasZoneOffset && offset == 0 && name != "UTC" && name != "GMT" && value.Location() != this.zone() {
			reason = "time zone abbreviation must be known in the configured location"
			causes = append(causes, fmt.Errorf("time zone abbreviation %q is unknown in %s", name, this.zone()))
			continue
		}
		if this.dateOnly {
			hour, minute, second := value.Clock()
			if hour != 0 || minute != 0 || second != 0 || value.Nanosecond() != 0 {
				reason = "date inputs must not include a nonzero time of day"
				continue
			}
		}
		return this.normalize(value), ctx.Err()
	}
	message := "a valid date and time expected"
	if this.dateOnly {
		message = "a valid date expected"
	}
	if reason != "" {
		message += "; " + reason
	}
	// Examples are derived solely from configured layouts, never from input.
	hour := 14
	if this.dateOnly {
		hour = 0
	}
	reference := time.Date(2026, time.July, 15, hour, 0, 0, 0, this.zone())
	var examples []string
	for _, layout := range this.layouts() {
		example := reference.Format(layout)
		if !slices.Contains(examples, example) {
			examples = append(examples, example)
		}
	}
	message += "; accepted formats (examples): " + strings.Join(examples, ", ")
	return time.Time{}, NewValidationError(message, causes...)
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
			return wire, NewMessageError("date/time location must be configured with WithLocation", ErrInvalid)
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
		return nil, NewMessageError("date/time zone offset is outside the supported range", err)
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return nil, NewMessageError("date/time configuration could not be encoded", err)
	}
	return []string{this.prefix() + string(encoded)}, ctx.Err()
}

func (this calendarCodec) parseWire(input string) (time.Time, error) {
	var wire calendarWire
	decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(input, this.prefix())))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return time.Time{}, NewValidationError("a valid date/time configuration expected", ErrIllegalArgument, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return time.Time{}, NewValidationError("date/time configuration must contain exactly one value", ErrIllegalArgument, err)
	}
	var value time.Time
	if err := value.UnmarshalBinary(wire.Time); err != nil {
		return time.Time{}, NewValidationError("a valid date/time configuration expected", ErrIllegalArgument, err)
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
		return time.Time{}, NewValidationError("a supported date/time configuration zone expected", ErrIllegalArgument)
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
