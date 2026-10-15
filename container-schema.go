package cli

import (
	"fmt"
	"reflect"
	"strings"
)

func errSchema(format string, args ...any) error { return fmt.Errorf(format, args...) }

type containerSchema struct {
	flags       []*Flag[any, any]
	flagByName  map[string]*Flag[any, any]
	flagByShort map[rune]*Flag[any, any]
	arguments   []*Argument[any, any]
	optionsType reflect.Type
}

func (this *containerSchema) addFlag(f *Flag[any, any]) error {
	if len(f.names) == 0 || f.names[0] == "" {
		return errSchema("flag needs a name")
	}
	names := make(map[string]bool)
	for _, name := range f.names {
		if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "= \t") {
			return errSchema("invalid flag name %q", name)
		}
		if this.flagByName[name] != nil || names[name] {
			return errSchema("duplicate flag name %q", name)
		}
		names[name] = true
	}
	shorts := make(map[rune]bool)
	for _, short := range f.shorts {
		if short == 0 || short == '-' || this.flagByShort[short] != nil || shorts[short] {
			return errSchema("duplicate or invalid short flag %q", short)
		}
		shorts[short] = true
	}
	for _, name := range f.names {
		this.flagByName[name] = f
	}
	for _, short := range f.shorts {
		this.flagByShort[short] = f
	}
	this.flags = append(this.flags, f)
	return nil
}

func (this *containerSchema) addArgument(a *Argument[any, any]) error {
	if a.name == "" {
		return errSchema("argument needs a name")
	}
	for _, prev := range this.arguments {
		if prev.name == a.name {
			return errSchema("duplicate argument name %q", a.name)
		}
	}
	a.index = len(this.arguments)
	this.arguments = append(this.arguments, a)
	return nil
}
