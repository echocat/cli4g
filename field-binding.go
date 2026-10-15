package cli

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// fieldBinding contains only type-level data resolved by Create. A [reflect.Value]
// for the actual field is obtained from this parse's options instance.
type fieldBinding struct {
	steps []fieldStep
	typ   reflect.Type
	path  []string // canonical names for the actual reflected index path
}

type fieldStep struct {
	index    int
	indirect bool
}

func bindField(typ reflect.Type, path []string, valueType reflect.Type, index *[]int) error {
	if len(path) <= 0 {
		return nil
	}
	if typ == nil {
		return errSchema("field %q requires an options type", strings.Join(path, "."))
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	for _, part := range path {
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct {
			return errSchema("field %q is not a struct field", strings.Join(path, "."))
		}
		field, ok := typ.FieldByName(part)
		if !ok {
			return errSchema("unknown options field %q", strings.Join(path, "."))
		}
		*index = append(*index, field.Index...)
		typ = field.Type
	}
	if typ != valueType {
		return errSchema("field %q has type %s, codec returns %s", strings.Join(path, "."), typ, valueType)
	}
	return nil
}

func bindUniqueField(typ reflect.Type, path []string, valueType reflect.Type, bound map[string]bool) ([]int, error) {
	var index []int
	if err := bindField(typ, path, valueType, &index); err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, nil
	}
	if err := registerFieldBinding(index, strings.Join(path, "."), bound); err != nil {
		return nil, err
	}
	return index, nil
}

// Exact keys preserve explicit-over-tag precedence. Descendant markers allow
// prefix checks in either registration order without scanning every binding.
func registerFieldBinding(index []int, name string, bound map[string]bool) error {
	key := fmt.Sprint(index)
	if bound[key] {
		return errSchema("options field %s is bound more than once", name)
	}
	if bound["descendant:"+key] {
		return errSchema("overlapping options field binding for %s: a descendant is already bound", name)
	}
	for depth := 1; depth < len(index); depth++ {
		if bound[fmt.Sprint(index[:depth])] {
			return errSchema("overlapping options field binding for %s: an ancestor is already bound", name)
		}
	}
	bound[key] = true
	for depth := 1; depth < len(index); depth++ {
		bound["descendant:"+fmt.Sprint(index[:depth])] = true
	}
	return nil
}

func compileFieldBinding(optionsType reflect.Type, index []int, expected reflect.Type) (*fieldBinding, error) {
	if len(index) == 0 {
		return nil, nil
	}
	if optionsType == nil {
		return nil, errSchema("options field requires an options type")
	}
	if optionsType.Kind() == reflect.Pointer {
		optionsType = optionsType.Elem()
	}
	result := &fieldBinding{steps: make([]fieldStep, 0, len(index)), typ: expected, path: make([]string, 0, len(index))}
	for n, i := range index {
		if optionsType.Kind() != reflect.Struct || i < 0 || i >= optionsType.NumField() {
			return nil, errSchema("invalid options field path %v", index)
		}
		field := optionsType.Field(i)
		result.path = append(result.path, field.Name)
		last := n == len(index)-1
		step := fieldStep{index: i}
		if last {
			// Scope-owned codec values must not acquire a second implicit owner
			// through Go's anonymous Close-method promotion on the options type.
			if field.Anonymous && (field.Type.Implements(reflect.TypeFor[io.Closer]()) || reflect.PointerTo(field.Type).Implements(reflect.TypeFor[io.Closer]())) {
				return nil, errSchema("anonymous codec binding %s promotes Close; use a named resource field", field.Name)
			}
			if !field.IsExported() {
				return nil, errSchema("options field %s must be exported", field.Name)
			}
			if field.Type != expected {
				return nil, errSchema("options field %s has type %s, codec returns %s", field.Name, field.Type, expected)
			}
		} else {
			// Unexported non-anonymous fields carry reflection's read-only flag
			// into their descendants. Private anonymous value embeddings do not.
			if !field.IsExported() && !field.Anonymous {
				return nil, errSchema("intermediate options field %s must be exported", field.Name)
			}
			next := field.Type
			if next.Kind() == reflect.Pointer {
				if !field.IsExported() {
					return nil, errSchema("pointer options field %s must be exported", field.Name)
				}
				step.indirect = true
				next = next.Elem()
			}
			if next.Kind() != reflect.Struct {
				return nil, errSchema("options field %s is not a struct", field.Name)
			}
			optionsType = next
		}
		result.steps = append(result.steps, step)
	}
	return result, nil
}

func (b *fieldBinding) set(root reflect.Value, value any) error {
	var v reflect.Value
	if value == nil {
		switch b.typ.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
			v = reflect.Zero(b.typ)
		default:
			return fmt.Errorf("codec returned nil for options field of type %s", b.typ)
		}
	} else {
		v = reflect.ValueOf(value)
		if !v.Type().AssignableTo(b.typ) {
			return fmt.Errorf("codec returned %s for options field of type %s", v.Type(), b.typ)
		}
	}
	dst, err := b.field(root)
	if err != nil {
		return err
	}
	dst.Set(v)
	return nil
}

func (b *fieldBinding) field(root reflect.Value) (reflect.Value, error) {
	dst := root
	if dst.Kind() == reflect.Pointer {
		dst = dst.Elem()
	}
	for _, step := range b.steps {
		dst = dst.Field(step.index)
		if step.indirect {
			if dst.IsNil() {
				if !dst.CanSet() {
					return reflect.Value{}, fmt.Errorf("options field is not settable")
				}
				dst.Set(reflect.New(dst.Type().Elem()))
			}
			dst = dst.Elem()
		}
	}
	if !dst.CanSet() {
		return reflect.Value{}, fmt.Errorf("options field is not settable")
	}
	return dst, nil
}
