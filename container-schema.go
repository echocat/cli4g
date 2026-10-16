package cli

import (
	"fmt"
	"reflect"
	"strings"
)

func errSchema(format string, args ...any) error { return fmt.Errorf(format, args...) }

type containerSchema struct {
	flags         []*Flag[any, any]
	flagByName    map[string]*Flag[any, any]
	flagByShort   map[rune]*Flag[any, any]
	flagVariants  map[string][]*Flag[any, any]
	shortVariants map[rune][]*Flag[any, any]
	arguments     []*Argument[any, any]
	optionsType   reflect.Type
}

func (this *containerSchema) addFlag(f *Flag[any, any]) error {
	if err := validateProfiles(f.profiles); err != nil {
		return err
	}
	if len(f.names) == 0 || f.names[0] == "" {
		return errSchema("flag needs a name")
	}
	names := make(map[string]bool)
	for _, name := range f.names {
		if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "= \t") {
			return errSchema("invalid flag name %q", name)
		}
		if names[name] {
			return errSchema("duplicate flag name %q", name)
		}
		for _, previous := range this.flagVariants[name] {
			if profilesOverlap(previous.profiles, f.profiles) {
				return errSchema("duplicate flag name %q (profiles %v and %v)", name, previous.profiles, f.profiles)
			}
		}
		names[name] = true
	}
	shorts := make(map[rune]bool)
	for _, short := range f.shorts {
		if short == 0 || short == '-' || shorts[short] {
			return errSchema("duplicate or invalid short flag %q", short)
		}
		for _, previous := range this.shortVariants[short] {
			if profilesOverlap(previous.profiles, f.profiles) {
				return errSchema("duplicate or invalid short flag %q (profiles %v and %v)", short, previous.profiles, f.profiles)
			}
		}
		shorts[short] = true
	}
	if this.flagVariants == nil {
		this.flagVariants = make(map[string][]*Flag[any, any])
		this.shortVariants = make(map[rune][]*Flag[any, any])
	}
	for _, name := range f.names {
		this.flagVariants[name] = append(this.flagVariants[name], f)
		if len(f.profiles) == 0 {
			this.flagByName[name] = f
		}
	}
	for _, short := range f.shorts {
		this.shortVariants[short] = append(this.shortVariants[short], f)
		if len(f.profiles) == 0 {
			this.flagByShort[short] = f
		}
	}
	this.flags = append(this.flags, f)
	return nil
}

func (this *containerSchema) addArgument(a *Argument[any, any]) error {
	if err := validateProfiles(a.profiles); err != nil {
		return err
	}
	if a.name == "" {
		return errSchema("argument needs a name")
	}
	for _, prev := range this.arguments {
		if prev.name == a.name && profilesOverlap(prev.profiles, a.profiles) {
			return errSchema("duplicate argument name %q", a.name)
		}
	}
	a.index = len(this.arguments)
	this.arguments = append(this.arguments, a)
	return nil
}
