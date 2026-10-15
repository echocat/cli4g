// Command report runs the comparison and renders the measured results as Markdown.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type measurement struct {
	nanoseconds float64
	bytes       float64
	allocs      float64
	characters  int
	complexity  int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "", "render existing Go benchmark output instead of measuring")
	output := flag.String("output", "../docs/benchmarks.md", "Markdown result path")
	raw := flag.String("raw", "results.txt", "raw benchmark output path")
	flag.Parse()
	var data []byte
	if *input != "" {
		var err error
		data, err = os.ReadFile(*input)
		if err != nil {
			return err
		}
	} else {
		var captured bytes.Buffer
		fmt.Fprintf(&captured, "measured: %s\ngo-version: %s\n", time.Now().UTC().Format(time.DateOnly), runtime.Version())
		command := exec.Command("go", "test", "-run=^$", "-bench=^BenchmarkLifecycle$", "-benchmem", "-benchtime=200ms", "-count=7", "-cpu=1", ".")
		command.Stdout = io.MultiWriter(os.Stdout, &captured)
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
		data = captured.Bytes()
		if err := os.WriteFile(*raw, data, 0o644); err != nil {
			return err
		}
	}
	results := make(map[string][]measurement)
	var cpu, measured, goVersion, goos, goarch string
	for _, line := range strings.Split(string(data), "\n") {
		for prefix, target := range map[string]*string{"measured:": &measured, "go-version:": &goVersion, "goos:": &goos, "goarch:": &goarch} {
			if strings.HasPrefix(line, prefix) {
				*target = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
		if strings.HasPrefix(line, "cpu:") {
			cpu = strings.TrimSpace(strings.TrimPrefix(line, "cpu:"))
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "BenchmarkLifecycle/") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(fields[0], "BenchmarkLifecycle/"), "-1")
		point := measurement{complexity: -1}
		for index := 2; index+1 < len(fields); index += 2 {
			value, err := strconv.ParseFloat(fields[index], 64)
			if err != nil {
				return err
			}
			switch fields[index+1] {
			case "ns/op":
				point.nanoseconds = value
			case "B/op":
				point.bytes = value
			case "allocs/op":
				point.allocs = value
			case "code-chars":
				if value <= 0 || value != float64(int(value)) {
					return fmt.Errorf("invalid code character count %q", fields[index])
				}
				point.characters = int(value)
			case "cognitive-complexity":
				if value < 0 || value != float64(int(value)) {
					return fmt.Errorf("invalid cognitive complexity %q", fields[index])
				}
				point.complexity = int(value)
			}
		}
		results[name] = append(results[name], point)
	}
	if len(results) != 54 {
		return fmt.Errorf("expected 54 benchmark cases; got %d", len(results))
	}
	if measured == "" || goVersion == "" || goos == "" || goarch == "" || cpu == "" {
		return fmt.Errorf("benchmark output is missing measurement metadata")
	}
	for name, points := range results {
		if _, err := median(points); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	var text strings.Builder
	fmt.Fprint(&text, "# CLI benchmark results\n\n")
	fmt.Fprintf(&text, "Measured %s with `%s`, `%s/%s`, `-cpu=1`, 200 ms per sample, seven samples.\n\n", measured, goVersion, goos, goarch)
	fmt.Fprintf(&text, "CPU: %s. Values below are sample medians. [Source and methodology](../_benchmarks) · [Raw results](../_benchmarks/results.txt).\n\n", cpu)
	fmt.Fprint(&text, "- **OneShot:** one definition + one parse/execution + cleanup.\n- **Reuse/N:** one definition + N parse/execution calls + cleanup; construction is timed.\n- `int32`, string, bool; identical rotating explicit/default inputs and checked handler results.\n- Library defaults for help, completion, version, output, and error responses are left unchanged; all measured inputs succeed and handlers produce no output.\n- Timings compare these adapters and workloads, not all features of each library.\n\n")
	fmt.Fprint(&text, "All timing columns use **⌀µs/op**, where one op is one parse/execution call. Each sample's batch time is divided by its call count (1, 100, or 10,000) and converted to microseconds, including amortized construction and cleanup. The reported value is the median of these per-sample averages.\n\n")
	fmt.Fprint(&text, "## Definition and binding modes\n\n- **Direct:** manually registered flags, read through accessors or standalone value pointers.\n- **StructTags:** options fields and input definitions discovered through reflection and struct tags; discovery and binding preparation are timed.\n\nCobra, urfave/cli, and Go `flag` do not provide native StructTags binding in the compared versions. `N/A` means the mode is not supported by the benchmarked native API.\n\n")
	fmt.Fprint(&text, "## Code characters\n\n**Code chars** counts non-whitespace Unicode characters inside each case's dedicated, gofmt-formatted `new…` body, including nested handler/run/reset/cleanup closures. All Unicode whitespace is excluded, including spaces, tabs, and newlines. Signatures, outer function braces, imports, separate struct definitions, and shared test/measurement code are excluded. Required supporting method bodies are included: Go `flag`'s `int32Value.String`/`Set` and Kong's `kongLeaf.Run`. Counts are stored with the timing samples and link to the readable adapter source. The same adapter supports all three lifecycles, so its code count is shown once.\n\n")
	fmt.Fprint(&text, "## Cognitive complexity\n\n**Cognitive complexity ↓** is calculated with [gocognit v1.3.0](https://github.com/uudashr/gocognit/tree/v1.3.0). Lower values indicate simpler control flow; zero is valid. The adapter score is the sum of the individual scores for the same constructor and required supporting methods used by Code chars. Nested closures are included once within their enclosing function, with gocognit's nesting rules. Analysis runs outside the timed region and scores are stored as `cognitive-complexity` in the raw samples. This measures control-flow complexity, not naming quality or overall readability.\n\n")
	modes := []struct {
		name      string
		libraries []string
	}{
		{"Direct", []string{"cli4g", "urfave", "cobra", "flag"}},
		{"StructTags", []string{"cli4g", "kong"}},
	}
	for _, shape := range []struct {
		name       string
		invocation string
	}{
		{"Root", "bench"},
		{"Subcommand", "bench a b"},
	} {
		for _, mode := range modes {
			fmt.Fprintf(&text, "## %s / %s\n\nInvocation: `%s [flags]`.\n\n", shape.name, mode.name, shape.invocation)
			fmt.Fprintln(&text, "| Library | Code chars | Cognitive complexity ↓ | OneShot ⌀µs/op | Reuse/100 ⌀µs/op | Reuse/10000 ⌀µs/op | OneShot B / allocs | Reuse/10000 B / allocs per invocation |")
			fmt.Fprintln(&text, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |")
			for _, library := range []string{"cli4g", "urfave", "cobra", "flag", "kong"} {
				if !slices.Contains(mode.libraries, library) {
					fmt.Fprintf(&text, "| %s | N/A | N/A | N/A | N/A | N/A | N/A | N/A |\n", library)
					continue
				}
				prefix := shape.name + "/" + mode.name + "/" + library + "/"
				one, err := median(results[prefix+"OneShot"])
				if err != nil {
					return fmt.Errorf("%s: %w", prefix, err)
				}
				hundred, err := median(results[prefix+"Reuse/N=100"])
				if err != nil {
					return err
				}
				tenThousand, err := median(results[prefix+"Reuse/N=10000"])
				if err != nil {
					return err
				}
				if one.characters != hundred.characters || one.characters != tenThousand.characters {
					return fmt.Errorf("%s: code character count differs between lifecycles", prefix)
				}
				if one.complexity != hundred.complexity || one.complexity != tenThousand.complexity {
					return fmt.Errorf("%s: cognitive complexity differs between lifecycles", prefix)
				}
				fmt.Fprintf(&text, "| %s | [%d](../_benchmarks/%s_test.go) | %d | %.2f | %.2f | %.2f | %.0f / %.0f | %.1f / %.2f |\n", library, one.characters, library, one.complexity, one.nanoseconds/1000, hundred.nanoseconds/(100*1000), tenThousand.nanoseconds/(10000*1000), one.bytes, one.allocs, tenThousand.bytes/10000, tenThousand.allocs/10000)
			}
			fmt.Fprintln(&text)
		}
	}
	fmt.Fprintln(&text, "## Scope\n\nVersions, adapter differences, reset policy, and reproduction commands are in the [benchmark module README](../_benchmarks). No I/O, help rendering, malformed inputs, parallel execution, or resource acquisition is measured. These are local measurements, not a portable performance guarantee.")
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(*output, []byte(text.String()), 0o644)
}

func median(points []measurement) (measurement, error) {
	if len(points) != 7 {
		return measurement{}, fmt.Errorf("expected seven samples; got %d", len(points))
	}
	for _, point := range points {
		if point.characters <= 0 || point.characters != points[0].characters {
			return measurement{}, fmt.Errorf("missing or inconsistent code character count")
		}
		if point.complexity < 0 || point.complexity != points[0].complexity {
			return measurement{}, fmt.Errorf("missing or inconsistent cognitive complexity")
		}
	}
	field := func(get func(measurement) float64) float64 {
		values := make([]float64, len(points))
		for index, point := range points {
			values[index] = get(point)
		}
		slices.Sort(values)
		return values[len(values)/2]
	}
	return measurement{
		nanoseconds: field(func(m measurement) float64 { return m.nanoseconds }),
		bytes:       field(func(m measurement) float64 { return m.bytes }),
		allocs:      field(func(m measurement) float64 { return m.allocs }),
		characters:  points[0].characters,
		complexity:  points[0].complexity,
	}, nil
}
