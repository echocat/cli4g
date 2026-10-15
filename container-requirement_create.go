package cli

import (
	"context"
	"fmt"
	"reflect"

	"github.com/echocat/cli4g/codec"
)

// schemaDraft is only used while Create is compiling command paths.
type schemaDraft struct {
	schema            containerSchema
	codecs            codec.Registry
	flagNames         map[*Flag[any, any]]string
	argumentNames     map[*Argument[any, any]]string
	flagFields        map[*Flag[any, any]][]int
	argumentFields    map[*Argument[any, any]][]int
	parentFields      []parentField
	argumentOverrides map[string]*Argument[any, any]
	taggedArguments   []*Argument[any, any]
}

func buildSchema(typ reflect.Type, cs []codec.Registration, fs []FlagRequirementLike, as []ArgumentRequirementLike) (schemaDraft, error) {
	s := schemaDraft{
		schema: containerSchema{
			optionsType: typ,
			flagByName:  make(map[string]*Flag[any, any]),
			flagByShort: make(map[rune]*Flag[any, any]),
		},
		flagNames:         make(map[*Flag[any, any]]string),
		flagFields:        make(map[*Flag[any, any]][]int),
		argumentNames:     make(map[*Argument[any, any]]string),
		argumentFields:    make(map[*Argument[any, any]][]int),
		argumentOverrides: make(map[string]*Argument[any, any]),
	}
	registry, err := codec.NewRegistry(context.Background(), cs)
	if err != nil {
		return s, err
	}
	s.codecs = registry
	if typ != nil {
		base := typ
		if base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		if base.Kind() != reflect.Struct {
			return s, errSchema("options type %s must be a struct or pointer to struct", typ)
		}
	}
	// Explicit field bindings take precedence over tags on that same field. Other
	// same-name definitions remain errors, including aliases and short names.
	bound := make(map[string]bool)
	for _, spec := range fs {
		if isNilRequirement(spec) {
			return s, errSchema("nil flag requirement")
		}
		f, err := spec.createRaw()
		if err != nil {
			return s, err
		}
		fieldIndex, err := bindUniqueField(typ, f.fieldPath, f.valueType, bound)
		if err != nil {
			return s, err
		}
		if len(fieldIndex) > 0 {
			s.flagFields[f] = fieldIndex
		}
		if err = s.schema.addFlag(f); err != nil {
			return s, err
		}
	}
	for _, spec := range as {
		if isNilRequirement(spec) {
			return s, errSchema("nil argument requirement")
		}
		a, err := spec.createRaw()
		if err != nil {
			return s, err
		}
		fieldIndex, err := bindUniqueField(typ, a.fieldPath, a.valueType, bound)
		if err != nil {
			return s, err
		}
		if len(fieldIndex) > 0 {
			s.argumentFields[a] = fieldIndex
			s.argumentOverrides[fmt.Sprint(fieldIndex)] = a
		}
		if err = s.schema.addArgument(a); err != nil {
			return s, err
		}
	}
	if typ != nil {
		base := typ
		if base.Kind() == reflect.Pointer {
			base = base.Elem()
		}
		if err := collectFields(base, nil, &s, bound); err != nil {
			return s, err
		}
	}
	if len(s.schema.arguments) > len(as) {
		// Only a genuinely mixed list needs tagged-slot ordering. Fully explicit
		// lists always keep registration order, regardless of any ignored tags.
		slots := make(map[*Argument[any, any]]bool, len(s.taggedArguments))
		for _, argument := range s.taggedArguments {
			slots[argument] = true
		}
		ordered := make([]*Argument[any, any], 0, len(s.schema.arguments))
		for _, argument := range s.schema.arguments[:len(as)] {
			if !slots[argument] {
				ordered = append(ordered, argument)
			}
		}
		ordered = append(ordered, s.taggedArguments...)
		s.schema.arguments = nil
		for _, argument := range ordered {
			if err := s.schema.addArgument(argument); err != nil {
				return s, err
			}
		}
	}
	return s, nil
}

func isNilRequirement(spec any) bool {
	if spec == nil {
		return true
	}
	value := reflect.ValueOf(spec)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
