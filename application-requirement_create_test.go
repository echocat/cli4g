package cli

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestCreateRejectsInheritedArityChanges(t *testing.T) {
	type flagOptions struct {
		Value string `flag:"value"`
	}
	type argumentOptions struct {
		Value string `argument:"value"`
	}
	t.Run("flag", func(t *testing.T) {
		assertInheritedArityRejected[*flagOptions](t, "inherited flag --value")
	})
	t.Run("argument", func(t *testing.T) {
		assertInheritedArityRejected[*argumentOptions](t, "inherited argument value")
	})
}

func assertInheritedArityRejected[O any](t *testing.T, message string) {
	t.Helper()
	_, err := NewApplication[O]().
		WithName("app").
		WithCommand(
			NewCommand[NoOptions]().
				WithName("child").
				WithCodec(grammarString{min: 2, max: 2}),
		).
		Create()
	require.ErrorContains(t, err, message)
	require.ErrorContains(t, err, "changes arity from 1,1 to 2,2")
}

func TestCreateRejectsInvalidArity(t *testing.T) {
	for _, arity := range []valueArity{
		{min: -1, max: 1},
		{min: 0, max: -2},
		{min: 2, max: 1},
	} {
		_, err := NewSimpleApplication().
			WithName("app").
			WithFlag(
				NewFlag(grammarString{min: arity.min, max: arity.max}).WithName("value"),
			).
			Create()
		require.ErrorContains(t, err, "invalid codec arity", "arity %v", arity)
	}
}

func TestCreateRejectsNonFinalVariadicArgument(t *testing.T) {
	_, err := NewSimpleApplication().
		WithName("app").
		WithArgument(
			NewArgument(codec.NewStrings()).WithName("values"),
			NewArgument(codec.NewString()).WithName("last"),
		).
		Create()
	require.ErrorContains(t, err, "only the final argument may be variadic")
}

func TestOptionsTypeIsDerivedFromGenericType(t *testing.T) {
	type options struct {
		Value string `flag:"value"`
	}
	valueRequirement := NewApplication[options]().
		WithName("value")
	assert.Equal(t, reflect.TypeFor[options](), valueRequirement.OptionsType())
	valueApp, err := valueRequirement.Create()
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeFor[options](), valueApp.OptionsType())

	pointerRequirement := NewApplication[*options]().
		WithName("pointer")
	assert.Equal(t, reflect.TypeFor[*options](), pointerRequirement.OptionsType())
	pointerApp, err := pointerRequirement.Create()
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeFor[*options](), pointerApp.OptionsType())
}

func TestCreateRejectsNonConcreteOptions(t *testing.T) {
	_, err := NewApplication[any]().
		WithName("interface").
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
	_, err = NewApplication[int]().
		WithName("scalar").
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
	_, err = NewSimpleApplication().
		WithName("app").
		WithCommand(NewCommand[any]().
			WithName("interface")).
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
}

func TestCreateRejectsNilCommandBuilder(t *testing.T) {
	var command *CommandRequirement[NoOptions]
	_, err := NewSimpleApplication().
		WithName("app").
		WithCommand(command).
		Create()
	require.ErrorContains(t, err, "nil command requirement")
}

func TestCommandsWithValueAndPointerBuilders(t *testing.T) {
	type childOptions struct {
		Value string `flag:"value"`
	}
	valueBuilder := *NewCommand[childOptions]().
		WithName("value")
	pointerBuilder := NewCommand[*childOptions]().
		WithName("pointer")
	app, err := NewSimpleApplication().
		WithName("app").
		WithCommand(valueBuilder, pointerBuilder).
		Create()
	require.NoError(t, err)
	command, found := app.Command("value")
	require.True(t, found)
	assert.Equal(t, reflect.TypeFor[childOptions](), command.OptionsType())
	command, found = app.Command("pointer")
	require.True(t, found)
	assert.Equal(t, reflect.TypeFor[*childOptions](), command.OptionsType())
}

func TestCreateRejectsCyclicCommandBuilders(t *testing.T) {
	pointer := NewSimpleCommand().
		WithName("pointer")
	pointer.WithCommand(pointer)
	_, err := NewSimpleApplication().
		WithName("app").
		WithCommand(pointer).
		Create()
	require.ErrorContains(t, err, "cyclic command requirement")

	children := make([]CommandRequirementLike, 1)
	value := CommandRequirement[NoOptions]{Names: []string{"value"}, Commands: children}
	children[0] = value
	_, err = NewSimpleApplication().
		WithName("app").
		WithCommand(value).
		Create()
	require.ErrorContains(t, err, "cyclic command requirement")
}

// Create benchmarks reuse a stable builder and include all schema compilation.
func BenchmarkApplicationCreate(b *testing.B) {
	for _, depth := range []int{0, 4, 16} {
		b.Run("CommandPath/Depth="+strconv.Itoa(depth), func(b *testing.B) {
			requirement := benchmarkCommandRequirement(depth)
			_, err := requirement.Create()
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := requirement.Create(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
