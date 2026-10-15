package codec

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrIllegalUsage = errors.New("illegal codec usage")
)

// Usage identifies flag input, positional input, or complete default configuration.
type Usage uint8

const (
	UsageFlag Usage = iota
	UsageArgument
	// UsageDefault decodes a complete default value, independent of the arity
	// of a single flag occurrence. In particular, slice defaults may be variadic.
	UsageDefault
)

func (this Usage) String() string {
	v, err := this.MarshalText()
	if err != nil {
		return fmt.Sprintf("illegal-codec-usage-%d", this)
	}
	return string(v)
}

func (this *Usage) Set(plain string) error {
	return this.UnmarshalText([]byte(plain))
}

func (this Usage) MarshalText() ([]byte, error) {
	switch this {
	case UsageFlag:
		return []byte("flag"), nil
	case UsageArgument:
		return []byte("argument"), nil
	case UsageDefault:
		return []byte("default"), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrIllegalUsage, this)
	}
}

func (this *Usage) UnmarshalText(v []byte) error {
	switch strings.ToLower(string(v)) {
	case "flag":
		*this = UsageFlag
		return nil
	case "argument":
		*this = UsageArgument
		return nil
	case "default":
		*this = UsageDefault
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrIllegalUsage, string(v))
	}
}
