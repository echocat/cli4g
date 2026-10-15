package codec

import (
	"fmt"
	"io"
	"reflect"
	"sync"
)

var requirementTypes sync.Map

type requirementPolicy struct{ valid, dynamic bool }

// Both compiled schemas and runtime validation consult the same type-only
// cache. Never cache requirement values or dynamic-interface validation results.
func policyForRequirement(typ reflect.Type) requirementPolicy {
	if cached, ok := requirementTypes.Load(typ); ok {
		return cached.(requirementPolicy)
	}
	policy := requirementPolicy{
		valid:   resourceFreeType(typ, map[reflect.Type]bool{}),
		dynamic: hasInterface(typ, map[reflect.Type]bool{}),
	}
	actual, _ := requirementTypes.LoadOrStore(typ, policy)
	return actual.(requirementPolicy)
}

// ValidateRequirement rejects observable resource-bearing/nonserializable requirement types once.
// Custom serializers must additionally obey the lossless, resource-free contract.
func ValidateRequirement(value any) error {
	if value == nil {
		return nil
	}
	typ := reflect.TypeOf(value)
	policy := policyForRequirement(typ)
	if !policy.valid || policy.dynamic && !resourceFreeValue(reflect.ValueOf(value), map[requirementVisit]bool{}) {
		return fmt.Errorf("%w: %s", ErrResourceRequirement, typ)
	}
	return nil
}

func hasInterface(typ reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[typ] {
		return false
	}
	seen[typ] = true
	switch typ.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer, reflect.Array, reflect.Slice:
		return hasInterface(typ.Elem(), seen)
	case reflect.Map:
		return hasInterface(typ.Key(), seen) || hasInterface(typ.Elem(), seen)
	case reflect.Struct:
		for field := range typ.Fields() {
			if hasInterface(field.Type, seen) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

type requirementVisit struct {
	typ    reflect.Type
	ptr    uintptr
	length int
}

func resourceFreeValue(value reflect.Value, seen map[requirementVisit]bool) bool {
	if !value.IsValid() {
		return true
	}
	if !policyForRequirement(value.Type()).valid {
		return false
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return true
		}
		return resourceFreeValue(value.Elem(), seen)
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if value.IsNil() {
			return true
		}
		// Reflection exposes identity only; no pointer is dereferenced or retained.
		visit := requirementVisit{typ: value.Type(), ptr: uintptr(value.UnsafePointer())}
		if value.Kind() == reflect.Slice {
			visit.length = value.Len()
		}
		if seen[visit] {
			return true
		}
		seen[visit] = true
		if value.Kind() == reflect.Pointer {
			return resourceFreeValue(value.Elem(), seen)
		}
		if value.Kind() == reflect.Map {
			iterator := value.MapRange()
			for iterator.Next() {
				if !resourceFreeValue(iterator.Key(), seen) || !resourceFreeValue(iterator.Value(), seen) {
					return false
				}
			}
			return true
		}
		for i := 0; i < value.Len(); i++ {
			if !resourceFreeValue(value.Index(i), seen) {
				return false
			}
		}
		return true
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if !resourceFreeValue(value.Index(i), seen) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for _, field := range value.Fields() {
			if !resourceFreeValue(field, seen) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func resourceFreeType(typ reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[typ] {
		return true
	}
	seen[typ] = true
	closer := reflect.TypeFor[io.Closer]()
	if typ.Implements(closer) || reflect.PointerTo(typ).Implements(closer) {
		return false
	}
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return resourceFreeType(typ.Elem(), seen)
	case reflect.Map:
		return resourceFreeType(typ.Key(), seen) && resourceFreeType(typ.Elem(), seen)
	case reflect.Interface:
		return true // dynamic values remain the implementer's contract
	case reflect.Struct:
		for field := range typ.Fields() {
			if !resourceFreeType(field.Type, seen) {
				return false
			}
		}
		return true
	default:
		return true
	}
}
