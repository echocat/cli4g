// Command coveragemerge unions Go coverage profiles before reporting them.
// A covered block stays covered across platforms, examples and shell E2E runs.
// Output uses set mode: hit counts represent execution at least once, not the
// sum of independent test runs. Relative example paths are restored to module
// paths so the Coveralls Go parser does not strip their directory components.
package main

import (
	"bufio"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const modulePath = "github.com/echocat/cli4g"

var coverageLine = regexp.MustCompile(`^(.+\.go):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)$`)

type position struct {
	file                                 string
	startLine, startCol, endLine, endCol int
}

type block struct {
	statements int
	hit        bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output, diagnostics io.Writer) error {
	flags := flag.NewFlagSet("coveragemerge", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	destination := flags.String("output", "profile.merged.cov", "merged Go set-mode profile")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() == 0 {
		return fmt.Errorf("at least one coverage profile is required")
	}
	blocks := make(map[position]block)
	for _, name := range flags.Args() {
		file, err := os.Open(name)
		if err != nil {
			return err
		}
		readErr := readProfile(file, blocks)
		closeErr := file.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	file, err := os.Create(*destination)
	if err != nil {
		return err
	}
	writeErr := writeProfile(file, blocks)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Merged %d coverage profiles into %s (%d unique blocks).\n", flags.NArg(), *destination, len(blocks))
	return err
}

func readProfile(input io.Reader, blocks map[position]block) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("empty coverage profile")
	}
	switch strings.TrimSpace(scanner.Text()) {
	case "mode: set", "mode: count", "mode: atomic":
	default:
		return fmt.Errorf("unsupported coverage mode")
	}
	count := 0
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		fields := coverageLine.FindStringSubmatch(line)
		if fields == nil {
			return fmt.Errorf("invalid coverage block")
		}
		name, err := sourcePath(fields[1])
		if err != nil {
			return err
		}
		var numbers [5]int
		for i := range numbers {
			numbers[i], err = strconv.Atoi(fields[i+2])
			if err != nil {
				return fmt.Errorf("invalid coverage coordinates: %w", err)
			}
		}
		pos := position{name, numbers[0], numbers[1], numbers[2], numbers[3]}
		if pos.startLine < 1 || pos.startCol < 1 || pos.endCol < 1 || pos.endLine < pos.startLine || pos.endLine == pos.startLine && pos.endCol < pos.startCol {
			return fmt.Errorf("invalid coverage coordinates")
		}
		hits, err := strconv.ParseUint(fields[7], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid coverage counter: %w", err)
		}
		current := blocks[pos]
		// Go patch versions can count composite-literal statements differently
		// for the same source span. Retain the largest count and union its hits.
		blocks[pos] = block{statements: max(current.statements, numbers[4]), hit: current.hit || hits > 0}
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("coverage profile has no blocks")
	}
	return nil
}

func sourcePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, modulePath+"/")
	name = path.Clean(strings.TrimPrefix(name, "./"))
	if name == "." || name == ".." || strings.HasPrefix(name, "../") || path.IsAbs(name) || strings.Contains(name, ":") {
		return "", fmt.Errorf("coverage source is outside the module: %s", name)
	}
	return modulePath + "/" + name, nil
}

func writeProfile(output io.Writer, blocks map[position]block) error {
	positions := make([]position, 0, len(blocks))
	for pos := range blocks {
		positions = append(positions, pos)
	}
	slices.SortFunc(positions, func(a, b position) int {
		return cmp.Or(cmp.Compare(a.file, b.file), cmp.Compare(a.startLine, b.startLine), cmp.Compare(a.startCol, b.startCol), cmp.Compare(a.endLine, b.endLine), cmp.Compare(a.endCol, b.endCol))
	})
	writer := bufio.NewWriter(output)
	if _, err := fmt.Fprintln(writer, "mode: set"); err != nil {
		return err
	}
	for _, pos := range positions {
		value := blocks[pos]
		hit := 0
		if value.hit {
			hit = 1
		}
		if _, err := fmt.Fprintf(writer, "%s:%d.%d,%d.%d %d %d\n", pos.file, pos.startLine, pos.startCol, pos.endLine, pos.endCol, value.statements, hit); err != nil {
			return err
		}
	}
	return writer.Flush()
}
