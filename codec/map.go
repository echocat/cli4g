package codec

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// MapKey restricts map keys to strings, booleans, and integers, including named
// types with these underlying types. Structs, pointers, floats, and complex
// numbers are not supported as keys.
type MapKey interface {
	~string | ~bool |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// NewMap composes a value codec into a repeatable map[K]V codec with map[K]R
// defaults. Each input is key=value, split at the first '='. Duplicate decoded
// keys are input errors, including across repeated flags. Flags take one pair
// per occurrence; arguments/defaults take zero or more. Map arguments must be
// last. Nil value codecs panic; nil/empty defaults stay distinct.
func NewMap[K MapKey, V, R any](element Codec[V, R]) Codec[map[K]V, map[K]R] {
	if isNil(element) {
		panic("map codec requires a value codec")
	}
	return mapCodec[K, V, R]{element: element}
}

type mapCodec[K MapKey, V, R any] struct{ element Codec[V, R] }

func (this mapCodec[K, V, R]) EnumChoices(ctx context.Context) ([]EnumChoice, error) {
	return DisplayEnum(ctx, this.element)
}

func (this mapCodec[K, V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	return Raw(ctx, this)
}

func (this mapCodec[K, V, R]) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if typ != reflect.TypeFor[map[K]V]() {
		return false, nil
	}
	return this.element.CanHandleType(ctx, reflect.TypeFor[V]())
}

func (mapCodec[K, V, R]) ExpectMinMax(_ context.Context, usage Usage) (int, int) {
	if usage == UsageFlag {
		return 1, 1
	}
	return 0, -1
}

func (mapCodec[K, V, R]) IsCumulative() bool { return true }

func (this mapCodec[K, V, R]) ParseRequirementInto(ctx context.Context, usage Usage, previous map[K]R, args ...string) (map[K]R, error) {
	additional, err := this.ParseRequirement(ctx, usage, args...)
	if err != nil {
		return previous, err
	}
	for _, key := range sortedMapKeys(additional) {
		if err := ctx.Err(); err != nil {
			return previous, err
		}
		if _, exists := previous[key]; exists {
			return previous, duplicateMapKey(key)
		}
	}
	if previous == nil {
		return additional, ctx.Err()
	}
	result := maps.Clone(previous)
	maps.Copy(result, additional)
	return result, ctx.Err()
}

// A structured representation preserves keys containing '=', arbitrary bytes,
// and values whose codecs serialize zero or multiple tokens.
type mapRequirementEntry struct {
	Key    []byte
	Values [][]byte
}

const mapRequirementPrefix = "map+json:"

func (this mapCodec[K, V, R]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (map[K]R, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return nil, err
	}
	if args == nil {
		return nil, nil
	}
	if usage == UsageDefault && len(args) == 1 && strings.HasPrefix(args[0], mapRequirementPrefix) {
		var entries []mapRequirementEntry
		if err := json.Unmarshal([]byte(strings.TrimPrefix(args[0], mapRequirementPrefix)), &entries); err != nil {
			return nil, fmt.Errorf("%w: map requirement: %w", ErrIllegalArgument, err)
		}
		if entries == nil {
			return nil, ctx.Err()
		}
		result := make(map[K]R, len(entries))
		for _, entry := range entries {
			var inputs []string
			if entry.Values != nil {
				inputs = make([]string, len(entry.Values))
				for i, input := range entry.Values {
					inputs[i] = string(input)
				}
			}
			if err := this.parseEntry(ctx, usage, result, string(entry.Key), inputs); err != nil {
				return nil, err
			}
		}
		return result, ctx.Err()
	}
	result := make(map[K]R, len(args))
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("%w: map entry must be key=value", ErrIllegalArgument)
		}
		if err := this.parseEntry(ctx, usage, result, key, []string{value}); err != nil {
			return nil, err
		}
	}
	return result, ctx.Err()
}

func (this mapCodec[K, V, R]) parseEntry(ctx context.Context, usage Usage, result map[K]R, input string, values []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key, err := parseMapKey[K](input)
	if err != nil {
		return fmt.Errorf("%w: map key %q: %w", ErrIllegalArgument, input, err)
	}
	if _, exists := result[key]; exists {
		return duplicateMapKey(key)
	}
	value, err := this.element.ParseRequirement(ctx, usage, values...)
	if err != nil {
		return fmt.Errorf("map value for key %q: %w", input, err)
	}
	result[key] = value
	return ctx.Err()
}

func (this mapCodec[K, V, R]) FormatRequirement(ctx context.Context, values map[K]R) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, nil
	}
	result := make([]string, 0, len(values))
	entries := make([]mapRequirementEntry, 0, len(values))
	structured := false
	for _, key := range sortedMapKeys(values) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plain, err := this.element.FormatRequirement(ctx, values[key])
		if err != nil {
			return nil, fmt.Errorf("map value for key %q: %w", formatMapKey(key), err)
		}
		name := formatMapKey(key)
		entry := mapRequirementEntry{Key: []byte(name)}
		if plain != nil {
			entry.Values = make([][]byte, len(plain))
			for i, input := range plain {
				entry.Values[i] = []byte(input)
			}
		}
		entries = append(entries, entry)
		if len(plain) != 1 || strings.Contains(name, "=") {
			structured = true
		} else {
			result = append(result, name+"="+plain[0])
		}
	}
	if len(result) == 1 && strings.HasPrefix(result[0], mapRequirementPrefix) {
		structured = true
	}
	if structured {
		encoded, err := json.Marshal(entries)
		if err != nil {
			return nil, err
		}
		return []string{mapRequirementPrefix + string(encoded)}, ctx.Err()
	}
	return result, ctx.Err()
}

func (this mapCodec[K, V, R]) DisplayRequirement(ctx context.Context, values map[K]R) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, nil
	}
	result := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plain, err := DisplayRequirement(ctx, this.element, values[key])
		if err != nil {
			return nil, fmt.Errorf("map value for key %q: %w", formatMapKey(key), err)
		}
		result = append(result, formatMapKey(key)+"="+strings.Join(plain, " "))
	}
	return result, ctx.Err()
}

func (this mapCodec[K, V, R]) CloneRequirement(ctx context.Context, values map[K]R) (map[K]R, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, nil
	}
	result := make(map[K]R, len(values))
	for _, key := range sortedMapKeys(values) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cloned, err := CloneRequirement(ctx, this.element, values[key])
		if err != nil {
			return nil, fmt.Errorf("map value for key %q: %w", formatMapKey(key), err)
		}
		result[key] = cloned
	}
	return result, ctx.Err()
}

func (this mapCodec[K, V, R]) Resolve(ctx context.Context, scope Scope, requirements map[K]R) (map[K]V, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if requirements == nil {
		return nil, nil
	}
	result := make(map[K]V, len(requirements))
	for _, key := range sortedMapKeys(requirements) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, err := this.element.Resolve(ctx, scope, requirements[key])
		if err != nil {
			return nil, fmt.Errorf("map value for key %q: %w", formatMapKey(key), err)
		}
		result[key] = value
	}
	return result, ctx.Err()
}

func parseMapKey[K MapKey](input string) (K, error) {
	var key K
	value := reflect.ValueOf(&key).Elem()
	switch value.Kind() {
	case reflect.String:
		value.SetString(input)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(input)
		if err != nil {
			return key, err
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(input, 0, value.Type().Bits())
		if err != nil {
			return key, err
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(input, 0, value.Type().Bits())
		if err != nil {
			return key, err
		}
		value.SetUint(parsed)
	}
	return key, nil
}

func formatMapKey[K MapKey](key K) string {
	value := reflect.ValueOf(key)
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	default:
		return strconv.FormatUint(value.Uint(), 10)
	}
}

func sortedMapKeys[K MapKey, V any](values map[K]V) []K {
	keys := slices.Collect(maps.Keys(values))
	slices.SortFunc(keys, func(a, b K) int { return strings.Compare(formatMapKey(a), formatMapKey(b)) })
	return keys
}

func duplicateMapKey[K MapKey](key K) error {
	return fmt.Errorf("%w: duplicate map key %q", ErrIllegalArgument, formatMapKey(key))
}
