package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func depthTestTree(levels int) (*CommandRequirement[NoOptions], []string) {
	var first, previous *CommandRequirement[NoOptions]
	var args []string
	for level := range levels {
		name := fmt.Sprintf("c%d", level+1)
		command := NewSimpleCommand(name)
		if previous == nil {
			first = command
		} else {
			previous.WithCommand(command)
		}
		previous = command
		args = append(args, name)
	}
	return first, args
}

func TestMaxCommandDepthChecksDefinitionAndSnapshotsLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit uint8
		depth int
		valid bool
	}{
		{"application only", 0, 0, true},
		{"zero rejects commands", 0, 1, false},
		{"one level", 1, 1, true},
		{"one rejects two", 1, 2, false},
		{"default boundary", 7, 7, true},
		{"default exceeded", 7, 8, false},
		{"custom boundary", 30, 30, true},
		{"custom exceeded", 30, 31, false},
		{"uint8 boundary", 255, 255, true},
		{"uint8 exceeded", 255, 256, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			requirement := NewSimpleApplication("app").WithMaxCommandDepth(test.limit)
			var args []string
			if test.depth > 0 {
				command, path := depthTestTree(test.depth)
				requirement.WithCommand(command)
				args = path
			}
			app, err := requirement.Create()
			if !test.valid {
				require.Error(t, err)
				assert.Contains(t, err.Error(), fmt.Sprintf("maximum command depth %d exceeded", test.limit))
				assert.Contains(t, err.Error(), "app "+strings.Join(args[:int(test.limit)+1], " "))
				assert.Nil(t, app)
				return
			}
			require.NoError(t, err)
			requirement.SetMaxCommandDepth(0)
			assert.Equal(t, test.limit, app.MaxCommandDepth())
			assert.Equal(t, uint8(test.depth), app.definition.commandDepth)
			invocation, err := app.Parse(nil, nil, args...)
			require.NoError(t, err)
			assert.Len(t, invocation.NameStack(), test.depth)
			assert.NoError(t, invocation.Close())
			assert.NoError(t, app.Close())
			assert.Equal(t, test.limit, app.MaxCommandDepth())
		})
	}
	assert.Equal(t, uint8(7), NewSimpleApplication().MaxCommandDepth)
	assert.Equal(t, uint8(7), NewApplication[NoOptions]().MaxCommandDepth)
}

func TestMaxCommandDepthIncludesInactiveProfilesButNotSiblingCount(t *testing.T) {
	_, err := NewSimpleApplication("app").WithMaxCommandDepth(0).
		WithCommand(NewSimpleCommand("inactive").WithProfiles("unused")).Create()
	assert.EqualError(t, err, `maximum command depth 0 exceeded at "app inactive"`)
	requirement := NewSimpleApplication("app").WithMaxCommandDepth(1)
	for i := range 40 {
		requirement.WithCommand(NewSimpleCommand(fmt.Sprintf("sibling%d", i)))
	}
	app := requirement.MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.NoError(t, app.ParseAndExecute(nil, nil, "sibling39"))
}

func TestScannerBuffersUseActualDepthAndTokenBound(t *testing.T) {
	root := NewSimpleApplication("app").WithMaxCommandDepth(255).WithStringFlag("value").MustCreate()
	defer func() { assert.NoError(t, root.Close()) }()
	input, err := root.scanTokens(context.Background(), nil, []string{"--value", "one"})
	require.NoError(t, err)
	assert.Equal(t, 1, cap(input.scopes), "a root-only definition must not reserve the configured maximum")
	assert.Equal(t, 1, cap(input.positions))
	tree, args := depthTestTree(3)
	app := NewSimpleApplication("app").WithMaxCommandDepth(30).WithCommand(tree).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	input, err = app.scanTokens(context.Background(), nil, args)
	require.NoError(t, err)
	assert.Equal(t, 4, cap(input.scopes))
	assert.Equal(t, 4, cap(input.positions))
	assert.Equal(t, 3, cap(input.commands))
	input, err = app.scanTokens(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, cap(input.scopes), "short input bounds scratch capacity even with a deeper definition")
}
