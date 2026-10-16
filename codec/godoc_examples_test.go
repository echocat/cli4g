package codec_test

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/echocat/cli4g/codec"
)

func ExampleNewDateTime() {
	value, err := codec.NewDateTime().
		ParseRequirement(context.Background(), codec.UsageFlag, "14 Oct 26 12:30 UTC")
	if err != nil {
		panic(err)
	}
	fmt.Println(value.Format(time.RFC3339))
	// Output: 2026-10-14T12:30:00Z
}

func ExampleDateCodec_WithFormats() {
	decoder := codec.NewDate().
		WithFormats("02.01.2006", time.DateOnly)
	value, err := decoder.ParseRequirement(context.Background(), codec.UsageArgument, "14.10.2026")
	if err != nil {
		panic(err)
	}
	fmt.Println(value.Format(time.DateOnly))
	// Output: 2026-10-14
}

func ExampleNewSlice() {
	decoder := codec.NewSlice(codec.NewDate())
	values, err := decoder.ParseRequirement(context.Background(), codec.UsageArgument, "2026-10-14", "15 Oct 26")
	if err != nil {
		panic(err)
	}
	for _, value := range values {
		fmt.Println(value.Format(time.DateOnly))
	}
	// Output:
	// 2026-10-14
	// 2026-10-15
}

func ExampleNewFile() {
	decoder := codec.NewFile(codec.FileWrite)
	requirement, err := decoder.ParseRequirement(context.Background(), codec.UsageFlag, "report.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println(requirement.Path, requirement.Create, requirement.Truncate)
	// Output: report.txt true false
}

func ExampleCombinedDefaults() {
	registry, err := codec.NewRegistry(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	if err := registry.AddDefaults(context.Background()); err != nil {
		panic(err)
	}
	bound, found, err := registry.Find(context.Background(), reflect.TypeFor[uint8](), "")
	if err != nil || !found {
		panic("no uint8 codec")
	}
	value, err := bound.ParseRequirement(context.Background(), codec.UsageFlag, "42")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%T: %v\n", value, value)
	// Output: uint8: 42
}

func ExampleDefaults() {
	original := codec.Defaults
	defer func() { codec.Defaults = original }()
	// Configure before creating applications, without concurrent modification.
	codec.Defaults = func(ctx context.Context) []codec.Codec[any, any] {
		return []codec.Codec[any, any]{codec.NewString().ToRaw(ctx)}
	}
	registry, err := codec.NewRegistry(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	if err := registry.AddDefaults(context.Background()); err != nil {
		panic(err)
	}
	_, found, err := registry.Find(context.Background(), reflect.TypeFor[string](), "")
	if err != nil {
		panic(err)
	}
	fmt.Println(found)
	// Output: true
}
