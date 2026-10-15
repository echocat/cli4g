package cli

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

func compilePaths(node *compileNode, parents []*compileNode, path string, owner *definitionLifecycle) error {
	// Subcommands are recognized only before the first positional value. A
	// required parent argument therefore cannot be supplied on a subcommand path.
	if len(node.children) > 0 {
		for _, argument := range node.draft.schema.arguments {
			if argument.IsRequired() {
				return fmt.Errorf("%s: required positional argument %q cannot be combined with subcommands", path, argument.Name())
			}
		}
	}
	stack := append(slices.Clone(parents), node)
	plan := &parsePlan{schemas: make([]containerSchema, len(stack)), definition: owner}
	for i, scope := range stack {
		bindings, err := compileParentBindings(scope.draft, stack[:i])
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if len(bindings) > 0 {
			if plan.parents == nil {
				plan.parents = &parentBindings{frames: make([][]parentBinding, len(stack))}
			}
			plan.parents.frames[i] = bindings
		}
		resolved, err := resolveSchema(scope.draft, stack, owner)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if i < len(parents) {
			original := scope.container.base().containerSchema
			if err := validateInheritedArity(original, resolved); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		plan.schemas[i] = resolved
	}
	definition := node.container.base()
	definition.containerSchema = plan.schemas[len(plan.schemas)-1]
	definition.plan = plan
	for _, child := range node.children {
		if err := compilePaths(child, stack, path+" "+child.container.Name(), owner); err != nil {
			return err
		}
	}
	return nil
}

func resolveSchema(draft schemaDraft, stack []*compileNode, owner *definitionLifecycle) (containerSchema, error) {
	result := containerSchema{
		optionsType: draft.schema.optionsType,
		flagByName:  make(map[string]*Flag[any, any]),
		flagByShort: make(map[rune]*Flag[any, any]),
	}
	for _, source := range draft.schema.flags {
		if source.required && source.HasDefault() {
			return result, fmt.Errorf("flag --%s: required cannot be combined with a default", source.Name())
		}
		flag := *source
		flag.names, flag.shorts, flag.envVars = slices.Clone(source.names), slices.Clone(source.shorts), slices.Clone(source.envVars)
		binding, err := compileFieldBinding(draft.schema.optionsType, draft.flagFields[source], source.valueType)
		if err != nil {
			return result, fmt.Errorf("flag --%s: %w", source.Name(), err)
		}
		flag.binding = binding
		if binding != nil {
			flag.fieldPath = binding.path
		}
		resolved, err := resolveDraftCodec(stack, source.codec, source.valueType, draft.flagNames[source])
		if err != nil {
			return result, fmt.Errorf("flag --%s: %w", source.Name(), err)
		}
		flag.codec = resolved
		flag.cumulative = codec.IsCumulative(resolved)
		flag.defaultValue, err = snapshotDefault(source.defaultValue, resolved, source.valueType, owner)
		if err != nil {
			return result, fmt.Errorf("flag --%s default: %w", source.Name(), err)
		}
		flag.arity, err = compileValueArity(context.Background(), resolved, codec.UsageFlag)
		if err != nil {
			return result, fmt.Errorf("flag --%s: %w", source.Name(), err)
		}
		if err := result.addFlag(&flag); err != nil {
			return result, err
		}
	}
	for i, source := range draft.schema.arguments {
		if source.required && source.HasDefault() {
			return result, fmt.Errorf("argument %s: required cannot be combined with a default", source.Name())
		}
		argument := *source
		binding, err := compileFieldBinding(draft.schema.optionsType, draft.argumentFields[source], source.valueType)
		if err != nil {
			return result, fmt.Errorf("argument %s: %w", source.Name(), err)
		}
		argument.binding = binding
		if binding != nil {
			argument.fieldPath = binding.path
		}
		resolved, err := resolveDraftCodec(stack, source.codec, source.valueType, draft.argumentNames[source])
		if err != nil {
			return result, fmt.Errorf("argument %s: %w", source.Name(), err)
		}
		argument.codec = resolved
		argument.defaultValue, err = snapshotDefault(source.defaultValue, resolved, source.valueType, owner)
		if err != nil {
			return result, fmt.Errorf("argument %s default: %w", source.Name(), err)
		}
		argument.arity, err = compileValueArity(context.Background(), resolved, codec.UsageArgument)
		if err != nil {
			return result, fmt.Errorf("argument %s: %w", source.Name(), err)
		}
		if i < len(draft.schema.arguments)-1 && argument.arity != (valueArity{min: 1, max: 1}) {
			return result, fmt.Errorf("argument %s: only the final argument may be variadic", source.Name())
		}
		if err := result.addArgument(&argument); err != nil {
			return result, err
		}
	}
	return result, nil
}

func validateInheritedArity(original, resolved containerSchema) error {
	for i, before := range original.flags {
		after := resolved.flags[i]
		if before.arity != after.arity {
			return fmt.Errorf("inherited flag --%s: codec override changes arity from %d,%d to %d,%d",
				before.Name(), before.arity.min, before.arity.max, after.arity.min, after.arity.max)
		}
	}
	for i, before := range original.arguments {
		after := resolved.arguments[i]
		if before.arity != after.arity {
			return fmt.Errorf("inherited argument %s: codec override changes arity from %d,%d to %d,%d",
				before.Name(), before.arity.min, before.arity.max, after.arity.min, after.arity.max)
		}
	}
	return nil
}

func resolveDraftCodec(stack []*compileNode, explicit codec.Codec[any, any], typ reflect.Type, name string) (codec.Codec[any, any], error) {
	if explicit != nil {
		return codec.Snapshot(context.Background(), explicit)
	}
	for i := len(stack) - 1; i >= 0; i-- {
		resolved, ok, err := stack[i].draft.codecs.Find(context.Background(), typ, name)
		if err != nil {
			return nil, err
		}
		if ok {
			return codec.Snapshot(context.Background(), resolved)
		}
	}
	if name != "" {
		return nil, fmt.Errorf("no codec for type %s with name %q", typ, name)
	}
	if resolved := fileCodecForType(typ, codec.NewFile()); resolved != nil {
		return resolved, nil
	}
	if resolved, found, err := codec.AutomaticFlagValue(context.Background(), typ); found || err != nil {
		return resolved, err
	}
	return nil, fmt.Errorf("no codec for type %s", typ)
}
