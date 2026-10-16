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
	_, err := NewApplication[O]("app").
		WithCommand(
			NewCommand[NoOptions]("child").
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
		_, err := NewSimpleApplication("app").
			WithFlag(
				NewFlag(grammarString{min: arity.min, max: arity.max}, "value"),
			).
			Create()
		require.ErrorContains(t, err, "invalid codec arity", "arity %v", arity)
	}
}

func TestCreateRejectsNonFinalVariadicArgument(t *testing.T) {
	_, err := NewSimpleApplication("app").
		WithArgument(
			NewArgument(codec.NewStrings(), "values"),
			NewArgument(codec.NewString(), "last"),
		).
		Create()
	require.ErrorContains(t, err, "only the final argument may be variadic")
}

func TestOptionsTypeIsDerivedFromGenericType(t *testing.T) {
	type options struct {
		Value string `flag:"value"`
	}
	valueRequirement := NewApplication[options]("value")
	assert.Equal(t, reflect.TypeFor[options](), valueRequirement.OptionsType())
	valueApp, err := valueRequirement.Create()
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeFor[options](), valueApp.OptionsType())

	pointerRequirement := NewApplication[*options]("pointer")
	assert.Equal(t, reflect.TypeFor[*options](), pointerRequirement.OptionsType())
	pointerApp, err := pointerRequirement.Create()
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeFor[*options](), pointerApp.OptionsType())
}

func TestCreateRejectsNonConcreteOptions(t *testing.T) {
	_, err := NewApplication[any]("interface").
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
	_, err = NewApplication[int]("scalar").
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
	_, err = NewSimpleApplication("app").
		WithCommand(NewCommand[any]("interface")).
		Create()
	require.ErrorContains(t, err, "must be a struct or pointer to struct")
}

func TestCreateRejectsNilCommandBuilder(t *testing.T) {
	var command *CommandRequirement[NoOptions]
	_, err := NewSimpleApplication("app").
		WithCommand(command).
		Create()
	require.ErrorContains(t, err, "nil command requirement")
}

func TestCommandsWithValueAndPointerBuilders(t *testing.T) {
	type childOptions struct {
		Value string `flag:"value"`
	}
	valueBuilder := *NewCommand[childOptions]("value")
	pointerBuilder := NewCommand[*childOptions]("pointer")
	app, err := NewSimpleApplication("app").
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
	pointer := NewSimpleCommand("pointer")
	pointer.WithCommand(pointer)
	_, err := NewSimpleApplication("app").
		WithCommand(pointer).
		Create()
	require.ErrorContains(t, err, "cyclic command requirement")

	children := make([]CommandRequirementLike, 1)
	value := CommandRequirement[NoOptions]{Names: []string{"value"}, Commands: children}
	children[0] = value
	_, err = NewSimpleApplication("app").
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
