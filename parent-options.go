package cli

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"slices"
)

type parentField struct {
	index     []int
	typ       reflect.Type
	mode      string
	anonymous bool
}
type parentBinding struct {
	ancestor int
	field    *fieldBinding
}
type parentBindings struct{ frames [][]parentBinding }

func collectParentField(field reflect.StructField, index []int, draft *schemaDraft) error {
	mode := field.Tag.Get("parent")
	if mode != "application" && mode != "matchingType" {
		return errSchema("field %s: invalid parent tag %q", field.Name, mode)
	}
	if field.Tag.Get("flag") != "" || field.Tag.Get("argument") != "" || field.Tag.Get("arg") != "" {
		return errSchema("parent field %s cannot also be a flag or argument", field.Name)
	}
	if !field.IsExported() {
		return errSchema("parent field %s must be exported", field.Name)
	}
	if field.Type.Kind() != reflect.Pointer || field.Type.Elem().Kind() != reflect.Struct {
		return errSchema("parent field %s must be a pointer to struct", field.Name)
	}
	for _, path := range draft.flagFields {
		if pathsOverlap(index, path) {
			return errSchema("parent field %s overlaps an explicit flag binding", field.Name)
		}
	}
	for _, path := range draft.argumentFields {
		if pathsOverlap(index, path) {
			return errSchema("parent field %s overlaps an explicit argument binding", field.Name)
		}
	}
	draft.parentFields = append(draft.parentFields, parentField{index: slices.Clone(index), typ: field.Type, mode: mode, anonymous: field.Anonymous})
	return nil
}
func pathsOverlap(a, b []int) bool {
	return slices.Equal(a[:min(len(a), len(b))], b[:min(len(a), len(b))])
}
func canonicalOptionsType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		return typ
	}
	return reflect.PointerTo(typ)
}
func compileParentBindings(draft schemaDraft, ancestors []*compileNode) ([]parentBinding, error) {
	var result []parentBinding
	for _, source := range draft.parentFields {
		ancestor := -1
		if source.mode == "application" {
			if len(ancestors) > 0 && canonicalOptionsType(ancestors[0].draft.schema.optionsType) == source.typ {
				ancestor = 0
			}
		} else {
			for i := len(ancestors) - 1; i >= 0; i-- {
				if canonicalOptionsType(ancestors[i].draft.schema.optionsType) == source.typ {
					ancestor = i
					break
				}
			}
		}
		if ancestor < 0 {
			return nil, errSchema("parent field %v: no %s ancestor with canonical options type %s", source.index, source.mode, source.typ)
		}
		if source.anonymous && (source.typ.Implements(reflect.TypeFor[Validatable]()) || source.typ.Implements(reflect.TypeFor[Initializable]()) || source.typ.Implements(reflect.TypeFor[io.Closer]())) {
			return nil, errSchema("anonymous parent field of type %s promotes lifecycle methods; use a named parent field", source.typ)
		}
		binding, err := compileFieldBinding(draft.schema.optionsType, source.index, source.typ)
		if err != nil {
			return nil, err
		}
		result = append(result, parentBinding{ancestor: ancestor, field: binding})
	}
	return result, nil
}

func (this *parentBindings) inject(ctx context.Context, index int, frames []invocationFrame) error {
	if this == nil || !frames[index].value.IsValid() {
		return nil
	}
	for _, binding := range this.frames[index] {
		// Attach borrowed cleanup dependencies even when a factory returned a
		// partial value/error or canceled. No user callback runs here. Report
		// cancellation only after these references are reachable by Options.Close.
		parent := frames[binding.ancestor].value
		if parent.Kind() != reflect.Pointer {
			parent = parent.Addr()
		}
		field, err := binding.field.field(frames[index].value)
		if err != nil {
			return fmt.Errorf("inject parent field %v: %w", binding.field.path, err)
		}
		if !field.IsNil() && field.Pointer() != parent.Pointer() {
			return fmt.Errorf("parent field %v has a conflicting factory-provided pointer", binding.field.path)
		}
		field.Set(parent)
	}
	return ctx.Err()
}
