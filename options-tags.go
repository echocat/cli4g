package cli

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

func collectFields(typ reflect.Type, prefix []int, s *schemaDraft, bound map[string]bool) error {
	return collectEmbeddedFields(typ, prefix, s, bound, nil)
}

func collectEmbeddedFields(typ reflect.Type, prefix []int, s *schemaDraft, bound map[string]bool, ancestors []reflect.Type) error {
	if slices.Contains(ancestors, typ) {
		return errSchema("recursive embedded options type %s", typ)
	}
	ancestors = append(ancestors, typ)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		idx := append(slices.Clone(prefix), i)
		if _, exists := field.Tag.Lookup("parent"); exists {
			if err := collectParentField(field, idx, s); err != nil {
				return err
			}
			continue // borrowed ancestor references never expose descendant tags
		}
		if bound[fmt.Sprint(idx)] {
			if replacement := s.argumentOverrides[fmt.Sprint(idx)]; replacement != nil &&
				(field.Tag.Get("argument") != "" || field.Tag.Get("arg") != "") {
				// Record only slots actually reachable through tag collection.
				// Named subobjects are explicit bindings, not implicit tag slots.
				s.taggedArguments = append(s.taggedArguments, replacement)
			}
			// A codec bound to the whole field owns its contents. Do not collect
			// descendant tags or recurse into an explicitly bound value type.
			continue
		}
		flagName := field.Tag.Get("flag")
		argName := field.Tag.Get("argument")
		if argName == "" {
			argName = field.Tag.Get("arg")
		}
		if flagName == "" && argName == "" {
			// Only unbound, untagged embeddings expose descendants. A whole-field
			// tag has the same opacity as an explicit WithField binding.
			embedded := field.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if field.Anonymous && embedded.Kind() == reflect.Struct {
				if err := collectEmbeddedFields(embedded, idx, s, bound, ancestors); err != nil {
					return err
				}
			}
			continue
		}
		if flagName != "" && argName != "" {
			return errSchema("field %s is both flag and argument", field.Name)
		}
		if !field.IsExported() {
			return errSchema("tagged field %s must be exported", field.Name)
		}
		if err := registerFieldBinding(idx, field.Name, bound); err != nil {
			return err
		}
		codecName := field.Tag.Get("type")
		fileCodec, err := fileTagCodec(field)
		if err != nil {
			return err
		}
		required, err := boolTag(field, "required", false)
		if err != nil {
			return err
		}
		if flagName != "" {
			visible, err := boolTag(field, "visible", true)
			if err != nil {
				return err
			}
			var shorts []rune
			for _, v := range splitTag(field.Tag.Get("short")) {
				r := []rune(v)
				if len(r) != 1 {
					return errSchema("invalid short flag %q on %s", v, field.Name)
				}
				shorts = append(shorts, r[0])
			}
			f := &Flag[any, any]{names: splitTag(flagName), shorts: shorts, envVars: splitTag(field.Tag.Get("env")),
				description: field.Tag.Get("description"), placeholder: field.Tag.Get("placeholder"),
				required: required, visible: visible, valueType: field.Type, codec: fileCodec}
			if value, exists := field.Tag.Lookup("default"); exists {
				f.defaultValue = &valueDefault[any]{strings: []string{value}, isString: true}
			}
			if err := s.schema.addFlag(f); err != nil {
				return err
			}
			s.flagNames[f] = codecName
			s.flagFields[f] = idx
		} else {
			a := &Argument[any, any]{name: argName, description: field.Tag.Get("description"),
				placeholder: field.Tag.Get("placeholder"), required: required,
				valueType: field.Type, codec: fileCodec}
			if value, exists := field.Tag.Lookup("default"); exists {
				a.defaultValue = &valueDefault[any]{strings: []string{value}, isString: true}
			}
			if err := s.schema.addArgument(a); err != nil {
				return err
			}
			s.argumentNames[a] = codecName
			s.argumentFields[a] = idx
			s.taggedArguments = append(s.taggedArguments, a)
		}
	}
	return nil
}

func splitTag(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "|")
}

func boolTag(f reflect.StructField, key string, def bool) (bool, error) {
	v := f.Tag.Get(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, errSchema("field %s: invalid %s tag %q", f.Name, key, v)
	}
	return b, nil
}
