package cli

import (
	"context"
	"os"
	"reflect"

	"github.com/echocat/cli4g/codec"
)

// Explicit file tags select a field-local codec. Without them, normal registry
// overrides remain available before the standard file fallback is considered.
func fileTagCodec(field reflect.StructField) (codec.Codec[any, any], error) {
	modeText, hasMode := field.Tag.Lookup("fileMode")
	_, hasSpecial := field.Tag.Lookup("allowSpecial")
	_, hasStdio := field.Tag.Lookup("allowStdio")
	if !hasMode && !hasSpecial && !hasStdio {
		return nil, nil
	}
	if field.Type != reflect.TypeFor[*os.File]() && field.Type != reflect.TypeFor[[]*os.File]() {
		return nil, errSchema("field %s: fileMode, allowSpecial and allowStdio require *os.File or []*os.File", field.Name)
	}
	if field.Tag.Get("type") != "" {
		return nil, errSchema("field %s: fileMode, allowSpecial and allowStdio cannot be combined with a named type codec", field.Name)
	}
	mode := codec.FileRead
	if hasMode {
		switch modeText {
		case "read":
		case "write":
			mode = codec.FileWrite
		case "readWrite":
			mode = codec.FileReadWrite
		default:
			return nil, errSchema("field %s: invalid fileMode tag %q; expected read, write or readWrite", field.Name, modeText)
		}
	}
	allowSpecial, err := boolTag(field, "allowSpecial", false)
	if err != nil {
		return nil, err
	}
	allowStdio, err := boolTag(field, "allowStdio", false)
	if err != nil {
		return nil, err
	}
	return fileCodecForType(field.Type, codec.NewFile(mode).WithAllowSpecial(allowSpecial).WithAllowStdio(allowStdio)), nil
}

func fileCodecForType(typ reflect.Type, source codec.FileCodec) codec.Codec[any, any] {
	ctx := context.Background()
	switch typ {
	case reflect.TypeFor[*os.File]():
		return source.ToRaw(ctx)
	case reflect.TypeFor[[]*os.File]():
		return codec.NewSlice(source).ToRaw(ctx)
	default:
		return nil
	}
}
