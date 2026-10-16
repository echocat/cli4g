package integration

func success(name string, args []string, stdout string) smokeCase {
	return smokeCase{name: name, args: args, want: expected{stdout: text(stdout)}}
}

func failure(name string, args []string, code int, diagnostic string, help bool) smokeCase {
	return smokeCase{name: name, args: args, want: expected{code: code, stdout: text(""), stderrContains: diagnostic, help: boolean(help)}}
}

func exampleCases() map[string][]smokeCase {
	return map[string][]smokeCase{
		"simple": {success("greeting", []string{"World"}, "Hello, World!\n")},
		"stdio": {
			success("injected streams", nil, "Injected input.\n"),
			success("uppercase", []string{"--uppercase"}, "INJECTED INPUT.\n"),
		},
		"argument-syntax":        {success("terminator", []string{"--", "-report.txt", "--help"}, "file=-report.txt more=[--help]\n")},
		"builder-fields":         {success("override", []string{"-g", "Hey", "Carl"}, "Hey, Carl!\n")},
		"commands":               {success("sum", []string{"sum", "2", "3", "4"}, "Sum: 9\n")},
		"commands-subcommands":   {success("product", []string{"math", "product", "2", "3", "4"}, "Product: 24\n")},
		"commands-inheritance":   {success("parent options", []string{"message", "greet", "Carl", "--greeting", "Hey", "--punctuation", "?"}, "Hey, Carl?\n")},
		"commands-introspection": {{name: "canonical alias", args: []string{"-v", "math", "add", "2", "3", "4"}, want: expected{stdout: text("Sum: 9\n"), stderr: text("Executing: math sum\n")}}},
		"custom-codec": {
			success("RGB", []string{"--color", "#FF8000"}, "red=255 green=128 blue=0\n"),
			failure("invalid color", []string{"--color", "invalid"}, 2, "color must be #RRGGBB", true),
		},
		"custom-help": {
			success("greeting", []string{"Carl"}, "Hello, Carl!\n"),
			{name: "unknown flag", args: []string{"--unknown"}, want: expected{code: 2, stderrContains: "unknown flag", help: boolean(true), helpStdout: true}},
		},
		"custom-resource-codec": {
			{name: "document", args: []string{"sample.txt"}, want: expected{stdout: text("Document contents.\n"), stderr: text("Open: sample.txt\nClose: sample.txt\n")}},
			failure("handler failure", []string{"sample.txt", "--fail"}, 1, "Close: sample.txt", false),
		},
		"custom-text-marshaller": {
			success("date", []string{"--date", "2026-10-13"}, "date=2026-10-13\n"),
			failure("invalid date", []string{"--date", "2026-02-30"}, 2, "date must be YYYY-MM-DD", true),
		},
		"custom-values": {
			success("map", []string{"-l", "debug", "-H", "Accept=text/plain"}, "level=debug headers=[Accept=text/plain]\n"),
			failure("invalid level", []string{"--level", "trace"}, 2, "level must be debug, info, warn, or error", true),
		},
		"date-time": {
			success("defaults", nil, "day=2026-10-14 at=2026-10-14T12:30:45.123456789Z until=2026-10-15T18:00:00Z reminders=[]\n"),
			success("formats and repeated dates", []string{"--day", "14 Oct 26", "--at", "14 Oct 26 14:30 +0200", "--reminder", "15.10.2026", "--reminder", "20261017", "--reminder", "15.10.2026", "16.10.2026 18:45"}, "day=2026-10-14 at=2026-10-14T14:30:00+02:00 until=2026-10-16T18:45:00Z reminders=[2026-10-15 2026-10-17 2026-10-15]\n"),
			success("date-only timestamp", []string{"--at", "20261016"}, "day=2026-10-14 at=2026-10-16T00:00:00Z until=2026-10-15T18:00:00Z reminders=[]\n"),
			success("custom RFC3339 layout", []string{"2026-10-16T20:45:00.987654321+02:00"}, "day=2026-10-14 at=2026-10-14T12:30:45.123456789Z until=2026-10-16T20:45:00.987654321+02:00 reminders=[]\n"),
			failure("invalid day", []string{"--day", "2026-02-30"}, 2, `cannot parse "2026-02-30"`, true),
			failure("timestamp rejected as date", []string{"--day", "2026-10-14T00:00:00Z"}, 2, `cannot parse "2026-10-14T00:00:00Z"`, true),
			failure("custom layouts replace defaults", []string{"2026-10-16 18:45:00"}, 2, `cannot parse "2026-10-16 18:45:00"`, true),
		},
		"embedded-options": {success("uppercase", []string{"echo", "-u", "Some text"}, "SOME TEXT\n")},
		"environment":      {{name: "CLI priority", args: []string{"--profile", "staging"}, env: []string{"CLI4G_PROFILE=production", "CLI4G_PORT=9090"}, want: expected{stdout: text("profile=staging port=9090\n")}}},
		"enums": {
			success("defaults", nil, "color=auto format=json level=info extra=0\n"),
			success("typed choices", []string{"--color", "never", "--format", "yaml", "--extra-format", "json", "debug"}, "color=never format=yaml level=debug extra=1\n"),
			failure("unknown format", []string{"--format", "toml"}, 2, `expected one of: "json", "yaml"`, true),
			failure("strict case", []string{"--format", "JSON"}, 2, "invalid enum value", true),
			failure("unknown level", []string{"trace"}, 2, `expected one of: "info", "debug"`, true),
			failure("unknown tag value", []string{"--color", "sometimes"}, 2, `expected one of: "auto", "always", "never"`, true),
		},
		"profiles": {
			{name: "text default", args: []string{"convert"}, env: []string{"CLI4G_PROFILES=text"}, want: expected{stdout: text("hello!\n")}},
			{name: "number variant", args: []string{"convert", "--foo", "7", "6"}, env: []string{"CLI4G_PROFILES=number"}, want: expected{stdout: text("42\n")}},
			{name: "text greeting", args: []string{"greet", "--principal", "Ada"}, env: []string{"CLI4G_PROFILES=text"}, want: expected{stdout: text("Hello, Ada\n")}},
			{name: "number greeting", args: []string{"greet", "--principal", "42"}, env: []string{"CLI4G_PROFILES=number"}, want: expected{stdout: text("Hello, #42\n")}},
			{name: "conflicting variants", env: []string{"CLI4G_PROFILES=text,number"}, want: expected{code: 2, stderrContains: `command "convert" is duplicated`, help: boolean(true)}},
			{name: "unknown profile", args: []string{"--help"}, env: []string{"CLI4G_PROFILES=typo"}, want: expected{code: 2, stderrContains: `unknown profile "typo"`, help: boolean(true)}},
		},
		"error-handling": {
			success("message", []string{"Hello"}, "Hello\n"),
			failure("unknown flag", []string{"--unknown"}, 64, "unknown flag", true),
			failure("handler failure", []string{"--fail"}, 70, "simulated execution failure", false),
		},
		"files": {success("read", []string{"sample.txt"}, "First line.\nSecond line.\n")},
		"lifecycle": {
			{name: "success", want: expected{stdout: text("Hello\n"), stderr: text("Initialize\nExecute\nClose\n")}},
			failure("initialize failure", []string{"--fail-initialize"}, 1, "Initialize\nClose\n", false),
			failure("handler failure", []string{"--fail"}, 1, "Initialize\nExecute\nClose\n", false),
		},
		"manual-invocation": {
			success("independent calls", nil, "Hello, World! (calls=1)\nHey, Carl! (calls=1)\n"),
			failure("unknown flag", []string{"--unknown"}, 2, "unknown flag", false),
		},
		"named-codecs": {
			success("policies", []string{"--interval", "250ms", "--delay", "2s"}, "interval=250ms delay=2s\n"),
			failure("zero interval", []string{"--interval", "0s"}, 2, "duration must be at least 1ns", true),
		},
		"options-factory":  {success("override", []string{"-g", "Hey", "Carl"}, "Hey, Carl!\n")},
		"repeatable-flags": {success("order and duplicates", []string{"-t", "blue", "-t", "green", "-t", "blue"}, "tags=[blue green blue]\n")},
		"typed-context":    {success("typed values", []string{"message", "greet", "Carl"}, "Hello, Carl! verbose=false\n")},
		"validation": {
			success("valid", []string{"-n", "2", "World"}, "Hello, World!\nHello, World!\n"),
			failure("invalid count", []string{"--count", "0", "World"}, 2, "count must be between 1 and 5", true),
		},
		"validation-errors": {
			success("valid", nil, "Accepted: 42\n"),
			failure("marked", []string{"invalid"}, 2, "Cause: strconv.NumError", true),
			failure("read failure", []string{"--fail-read"}, 1, "Cause: io.ErrUnexpectedEOF", false),
		},
		"cancellation": {failure("zero sleep", []string{"--sleep", "0s"}, 2, "sleep must be greater than zero", true)},
	}
}
