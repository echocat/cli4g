package benchmarks

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/echocat/cli4g/_benchmarks/internal/sourcecode"
)

type values struct {
	Count   int32
	Name    string
	Verbose bool
}

type recorder struct {
	want  values
	calls int
}

func (result *recorder) accept(got values) error {
	if got != result.want {
		return fmt.Errorf("decoded values: got %+v, want %+v", got, result.want)
	}
	result.calls++
	return nil
}

type runner struct {
	run   func(context.Context, []string) error
	close func() error
}

func (definition runner) Close() error {
	if definition.close != nil {
		return definition.close()
	}
	return nil
}

type factory struct {
	shape   string
	mode    string
	library string
	new     func(*recorder) (runner, error)
}

var factories = []factory{
	{"Root", "Direct", "cli4g", newCLI4GRootDirect},
	{"Root", "StructTags", "cli4g", newCLI4GRootStructTags},
	{"Root", "Direct", "urfave", newUrfaveRootDirect},
	{"Root", "PointerBindings", "urfave", newUrfaveRootPointerBindings},
	{"Root", "Direct", "cobra", newCobraRootDirect},
	{"Root", "PointerBindings", "cobra", newCobraRootPointerBindings},
	{"Root", "Direct", "flag", newFlagRootDirect},
	{"Root", "PointerBindings", "flag", newFlagRootPointerBindings},
	{"Root", "StructTags", "kong", newKongRootStructTags},
	{"Subcommand", "Direct", "cli4g", newCLI4GSubcommandDirect},
	{"Subcommand", "StructTags", "cli4g", newCLI4GSubcommandStructTags},
	{"Subcommand", "Direct", "urfave", newUrfaveSubcommandDirect},
	{"Subcommand", "PointerBindings", "urfave", newUrfaveSubcommandPointerBindings},
	{"Subcommand", "Direct", "cobra", newCobraSubcommandDirect},
	{"Subcommand", "PointerBindings", "cobra", newCobraSubcommandPointerBindings},
	{"Subcommand", "Direct", "flag", newFlagSubcommandDirect},
	{"Subcommand", "PointerBindings", "flag", newFlagSubcommandPointerBindings},
	{"Subcommand", "StructTags", "kong", newKongSubcommandStructTags},
}

func (entry factory) name() string {
	return entry.shape + "/" + entry.mode + "/" + entry.library
}

func (entry factory) codeMetrics() (sourcecode.Metrics, error) {
	symbol := runtime.FuncForPC(reflect.ValueOf(entry.new).Pointer()).Name()
	names := []string{symbol[strings.LastIndex(symbol, ".")+1:]}
	switch entry.library {
	case "flag":
		names = append(names, "int32Value.String", "int32Value.Set")
	case "kong":
		names = append(names, "kongLeaf.Run")
	}
	return sourcecode.AnalyzeFile(entry.library+"_test.go", names...)
}

type input struct {
	args []string
	want values
}

func inputs(shape string) []input {
	path := []string{"bench"}
	if shape == "Subcommand" {
		path = []string{"bench", "a", "b"}
	}
	return []input{
		{append(append([]string(nil), path...), "--count=7", "--name=alice", "--verbose=true"), values{7, "alice", true}},
		{append(append([]string(nil), path...), "--count=-3", "--name=bob", "--verbose=false"), values{-3, "bob", false}},
		{append([]string(nil), path...), values{5, "default", false}},
	}
}

func TestAdapters(t *testing.T) {
	for _, entry := range factories {
		t.Run(entry.name(), func(t *testing.T) {
			if _, err := entry.codeMetrics(); err != nil {
				t.Fatal(err)
			}
			result := new(recorder)
			definition, err := entry.new(result)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := definition.Close(); err != nil {
					t.Error(err)
				}
			}()
			for range 3 {
				for _, input := range inputs(entry.shape) {
					result.want = input.want
					before := result.calls
					if err := definition.run(t.Context(), input.args); err != nil {
						t.Fatal(err)
					}
					if result.calls != before+1 {
						t.Fatal("handler must execute exactly once")
					}
				}
			}
		})
	}
}

func BenchmarkLifecycle(b *testing.B) {
	for _, entry := range factories {
		metrics, err := entry.codeMetrics()
		if err != nil {
			b.Fatal(err)
		}
		for _, executions := range []int{1, 100, 10000} {
			lifecycle := "OneShot"
			if executions > 1 {
				lifecycle = "Reuse/N=" + strconv.Itoa(executions)
			}
			b.Run(entry.name()+"/"+lifecycle, func(b *testing.B) {
				cases := inputs(entry.shape)
				result := new(recorder)
				ctx := context.Background()
				b.ReportAllocs()
				iteration := 0
				for b.Loop() {
					definition, err := entry.new(result)
					if err != nil {
						b.Fatal(err)
					}
					for index := range executions {
						input := cases[(iteration*executions+index)%len(cases)]
						result.want = input.want
						if err := definition.run(ctx, input.args); err != nil {
							b.Fatal(err)
						}
					}
					if err := definition.Close(); err != nil {
						b.Fatal(err)
					}
					iteration++
				}
				b.StopTimer()
				if result.calls != b.N*executions {
					b.Fatalf("handler calls: got %d, want %d", result.calls, b.N*executions)
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*executions), "ns/invocation")
				b.ReportMetric(float64(metrics.Characters), "code-chars")
				b.ReportMetric(float64(metrics.CognitiveComplexity), "cognitive-complexity")
			})
		}
	}
}
