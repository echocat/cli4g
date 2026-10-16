package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRejectsTypedNilFlagAndArgumentRequirements(t *testing.T) {
	var flag *FlagRequirement[string, string]
	var argument *ArgumentRequirement[string, string]
	for _, scope := range []string{"application", "command"} {
		for _, kind := range []string{"flag", "argument"} {
			for _, typed := range []bool{false, true} {
				t.Run(scope+"/"+kind+map[bool]string{false: "/nil-interface", true: "/typed-nil"}[typed], func(t *testing.T) {
					root := NewSimpleApplication("app")
					child := NewSimpleCommand("run")
					if kind == "flag" {
						var spec FlagRequirementLike
						if typed {
							spec = flag
						}
						if scope == "application" {
							root.WithFlag(spec)
						} else {
							child.WithFlag(spec)
						}
					} else {
						var spec ArgumentRequirementLike
						if typed {
							spec = argument
						}
						if scope == "application" {
							root.WithArgument(spec)
						} else {
							child.WithArgument(spec)
						}
					}
					if scope == "command" {
						root.WithCommand(child)
					}
					var err error
					var app *Application[NoOptions]
					require.NotPanics(t, func() { app, err = root.Create() })
					assert.Nil(t, app)
					require.ErrorContains(t, err, "nil "+kind+" requirement")
				})
			}
		}
	}
}
