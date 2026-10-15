package sourcecode

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCountBodyCharacters(t *testing.T) {
	source := []byte(`package example

type Options struct {
	Value string
}

func newExample(ignored Options) string {
	message := "Grüße  Welt"
	return message
}

func (options *Options) Run() string {
	return options.Value
}
`)
	constructor := "message:=\"GrüßeWelt\"returnmessage"
	handler := "returnoptions.Value"
	for _, test := range []struct {
		name  string
		input []byte
		names []string
		body  string
	}{
		{"constructor only", source, []string{"newExample"}, constructor},
		{"required handler", source, []string{"newExample", "Options.Run"}, constructor + handler},
		{"CRLF normalized", []byte(strings.ReplaceAll(string(source), "\n", "\r\n")), []string{"newExample"}, constructor},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := analyzeSource(test.input, test.names...)
			if err != nil {
				t.Fatal(err)
			}
			if want := utf8.RuneCountInString(test.body); got.Characters != want {
				t.Fatalf("characters: got %d, want %d", got.Characters, want)
			}
		})
	}
}

func TestCountRejectsMissingBody(t *testing.T) {
	if _, err := analyzeSource([]byte("package example\nfunc other() {}\n"), "newExample"); err == nil {
		t.Fatal("missing constructor must not silently produce zero characters")
	}
}

func TestCountIncludesNestedRunClosure(t *testing.T) {
	body := "\n\treturn func() string {\n\t\treturn \"run\"\n\t}\n"
	source := []byte("package example\nfunc newExample() func() string {" + body + "}\n")
	got, err := analyzeSource(source, "newExample")
	if err != nil {
		t.Fatal(err)
	}
	if want := utf8.RuneCountInString("returnfunc()string{return\"run\"}"); got.Characters != want {
		t.Fatalf("characters: got %d, want %d", got.Characters, want)
	}
}

func TestCognitiveComplexityScope(t *testing.T) {
	source := []byte(`package example

type Options struct{}

func newExample() func(bool) int {
	return func(enabled bool) int {
		if enabled {
			return 1
		}
		return 0
	}
}

func (options *Options) Run(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

func unrelated(enabled bool) {
	for enabled {
		if enabled {
			break
		}
	}
}

func straightLine() int { return 1 }
`)
	for _, test := range []struct {
		name  string
		names []string
		want  int
	}{
		{"nested closure counted once", []string{"newExample"}, 2},
		{"supporting method included", []string{"newExample", "Options.Run"}, 3},
		{"zero is valid", []string{"straightLine"}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := analyzeSource(source, test.names...)
			if err != nil {
				t.Fatal(err)
			}
			if got.CognitiveComplexity != test.want {
				t.Fatalf("cognitive complexity: got %d, want %d", got.CognitiveComplexity, test.want)
			}
		})
	}
}
