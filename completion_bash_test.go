package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bashForCompletionTest(t *testing.T) string {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("Bash is not installed")
	}
	if err := exec.Command(bash, "--noprofile", "--norc", "-c", "((BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 4)))").Run(); err != nil {
		t.Skip("Bash 4.4 or newer is required")
	}
	return bash
}

func TestBashCompletionQuoteContextDoesNotEvaluateInput(t *testing.T) {
	bash := bashForCompletionTest(t)
	app := NewSimpleApplication("tool").WithCompletion(true).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	script, err := app.CompletionScript(CompletionBash)
	require.NoError(t, err)
	// A script file avoids Windows/MSYS argument decoding of a large -c string.
	// Pass test inputs through the environment to preserve quotes/backslashes too.
	path := filepath.Join(t.TempDir(), "quote-context.bash")
	require.NoError(t, os.WriteFile(path, []byte(script+`
COMP_LINE="$CLI4G_TEST_LINE"
COMP_POINT="$CLI4G_TEST_POINT"
[[ $(complete -p tool) =~ -F\ ([^\ ]+) ]] || exit 1
quote=
"${BASH_REMATCH[1]}_quote_context" || exit 1
printf '%s' "$quote"
`), 0600))
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
			command := exec.Command(bash, "--noprofile", "--norc", path)
			command.Env = append(os.Environ(), "CLI4G_TEST_LINE="+test.line, "CLI4G_TEST_POINT="+strconv.Itoa(len(test.line)))
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			assert.Equal(t, test.want, string(output))
		})
	}
}
