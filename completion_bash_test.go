package cli

import (
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashCompletionQuoteContextDoesNotEvaluateInput(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("Bash is not installed")
	}
	app := NewSimpleApplication("tool").WithCompletion(true).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	script, err := app.CompletionScript(CompletionBash)
	require.NoError(t, err)
	for _, test := range []struct{ line, want string }{
		{`tool --value plain`, ""},
		{`tool --value '`, "'"},
		{`tool --value "`, `"`},
		{`tool --value ''`, ""},
		{`tool --value ""`, ""},
		{`tool --value \'`, ""},
		{`tool --value "a\"`, `"`},
		{`tool --value "a\q"`, ""},
		{`tool --value '$HOME`, "'"},
		{`tool --value "$(printf literal)`, `"`},
		{`tool --value "\` + "`" + `literal`, `"`},
		{`tool --context 'über😀' --value "`, `"`},
		{`tool --value $'`, "ansi"},
		{`tool --value $'a\'`, "ansi"},
		{`tool $'a\'' --value "`, `"`},
		{`tool --value \$'`, "'"},
	} {
		t.Run(test.line, func(t *testing.T) {
			command := exec.Command("bash", "--noprofile", "--norc", "-c", script+`
COMP_LINE="$1"
COMP_POINT="$2"
[[ $(complete -p tool) =~ -F\ ([^\ ]+) ]]
quote=
"${BASH_REMATCH[1]}_quote_context"
printf '%s' "$quote"
`, "--", test.line, strconv.Itoa(len(test.line)))
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.Equal(t, test.want, string(output))
		})
	}
}
