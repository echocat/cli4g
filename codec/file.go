package codec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
)

// FileMode selects read, write, or read/write access for file inputs.
type FileMode uint8

const (
	FileRead FileMode = iota
	FileWrite
	FileReadWrite
)

// FileRequirement describes how to open a file; typed defaults choose all settings.
// Append and Truncate require write access and cannot be combined. Permission=0
// means 0600 for newly created files, subject to umask. Paths are literal unless
// AllowStdio enables exact aliases: "-"/"stdin" for FileRead and "-"/"stdout"
// for FileWrite. Stdio aliases reject read/write access, mismatched directions,
// Append and Truncate; Create and Permission have no effect on them.
// Create/Truncate can affect files before options validation; closing is not rollback.
type FileRequirement struct {
	Path         string      `json:"path"`
	Mode         FileMode    `json:"mode"`
	Create       bool        `json:"create"`
	Append       bool        `json:"append"`
	Truncate     bool        `json:"truncate"`
	Permission   fs.FileMode `json:"permission"`
	AllowSpecial bool        `json:"allowSpecial"`
	AllowStdio   bool        `json:"allowStdio"`
}

var ErrSpecialFile = NewValidationError(errors.New("special files require explicit opt-in"))

// ErrStdioUnavailable marks a stdio alias without a usable *os.File in its context.
var ErrStdioUnavailable = errors.New("stdio file stream unavailable")

// FileCodec resolves paths to invocation-owned *os.File handles or borrowed
// standard streams when [FileCodec.WithAllowStdio] is enabled.
// Filesystem paths must name regular files unless [FileCodec.WithAllowSpecial]
// is enabled.
// Help and default inspection do not open files. Use named options fields:
// anonymous io.Closer embeddings would transfer cleanup to two owners.
type FileCodec struct {
	mode         FileMode
	allowSpecial bool
	allowStdio   bool
}

// NewFile creates a file codec; omitted mode means [FileRead].
// Write modes create missing files but do not truncate existing contents.
// Passing more than one mode panics. Typed [FileRequirement] values specify their
// own complete policy, independent of the codec's path-input defaults.
func NewFile(mode ...FileMode) FileCodec {
	if len(mode) > 1 {
		panic("file codec accepts at most one mode")
	}
	selected := FileRead
	if len(mode) == 1 {
		selected = mode[0]
	}
	return FileCodec{mode: selected}
}

// WithAllowSpecial permits directories, devices, FIFOs, and other nonregular paths.
// Native file operations may block despite cancellation, including ordinary files.
// Typed [FileRequirement] values choose their own AllowSpecial policy.
func (this FileCodec) WithAllowSpecial(allow bool) FileCodec { this.allowSpecial = allow; return this }

// WithAllowStdio interprets "-"/"stdin" as context stdin in FileRead mode and
// "-"/"stdout" as context stdout in FileWrite mode, independently of AllowSpecial.
// These streams are borrowed and never closed by the scope. Use ./stdin,
// ./stdout or ./- for literal paths. Typed [FileRequirement] values choose
// their own AllowStdio policy. Streams must be non-nil *os.File values;
// missing or incompatible streams return [ErrStdioUnavailable]. No implicit
// process-stdio fallback is performed by the codec.
func (this FileCodec) WithAllowStdio(allow bool) FileCodec { this.allowStdio = allow; return this }

func (this FileCodec) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (FileCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[*os.File](), ctx.Err()
}

func (FileCodec) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this FileCodec) ParseRequirement(ctx context.Context, usage Usage, args ...string) (FileRequirement, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return FileRequirement{}, err
	}
	requirement := FileRequirement{Path: args[0], Mode: this.mode, Create: this.mode != FileRead, AllowSpecial: this.allowSpecial, AllowStdio: this.allowStdio}
	if usage == UsageDefault && strings.HasPrefix(args[0], "file+json:") {
		var wire fileRequirementWire
		decoder := json.NewDecoder(strings.NewReader(strings.TrimPrefix(args[0], "file+json:")))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&wire); err != nil {
			return FileRequirement{}, fmt.Errorf("%w: file requirement: %w", ErrIllegalArgument, err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return FileRequirement{}, fmt.Errorf("%w: trailing file requirement data", ErrIllegalArgument)
		}
		requirement = wire.requirement()
	}
	return requirement, validateFileRequirement(requirement)
}

func (FileCodec) FormatRequirement(ctx context.Context, requirement FileRequirement) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateFileRequirement(requirement); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(fileRequirementWire{Path: []byte(requirement.Path), Mode: requirement.Mode, Create: requirement.Create, Append: requirement.Append, Truncate: requirement.Truncate, Permission: requirement.Permission, AllowSpecial: requirement.AllowSpecial, AllowStdio: requirement.AllowStdio})
	if err != nil {
		return nil, err
	}
	return []string{"file+json:" + string(encoded)}, nil
}

// []byte JSON serialization preserves non-UTF-8 paths through base64 encoding.
// No live value or OS metadata is used when serializing requirements.
type fileRequirementWire struct {
	Path         []byte      `json:"path"`
	Mode         FileMode    `json:"mode"`
	Create       bool        `json:"create"`
	Append       bool        `json:"append"`
	Truncate     bool        `json:"truncate"`
	Permission   fs.FileMode `json:"permission"`
	AllowSpecial bool        `json:"allowSpecial"`
	AllowStdio   bool        `json:"allowStdio"`
}

func (wire fileRequirementWire) requirement() FileRequirement {
	return FileRequirement{Path: string(wire.Path), Mode: wire.Mode, Create: wire.Create, Append: wire.Append, Truncate: wire.Truncate, Permission: wire.Permission, AllowSpecial: wire.AllowSpecial, AllowStdio: wire.AllowStdio}
}

func (FileCodec) DisplayRequirement(ctx context.Context, requirement FileRequirement) ([]string, error) {
	if err := validateFileRequirement(requirement); err != nil {
		return nil, err
	}
	return []string{requirement.Path}, ctx.Err()
}

func (FileCodec) CloneRequirement(ctx context.Context, requirement FileRequirement) (FileRequirement, error) {
	if err := validateFileRequirement(requirement); err != nil {
		return FileRequirement{}, err
	}
	return requirement, ctx.Err()
}

func (FileCodec) Resolve(ctx context.Context, scope Scope, requirement FileRequirement) (*os.File, error) {
	if isNil(scope) {
		return nil, fmt.Errorf("%w: file resolution requires a scope", ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateFileRequirement(requirement); err != nil {
		return nil, err
	}
	if requirement.AllowStdio && isStdioPath(requirement.Path) {
		var stream any
		name := "stdout"
		if requirement.Mode == FileRead {
			name, stream = "stdin", StdinOfContext(ctx)
		} else {
			stream = StdoutOfContext(ctx)
		}
		file, ok := stream.(*os.File)
		if !ok || file == nil {
			return nil, fmt.Errorf("%w: %s alias %q requires a non-nil *os.File in context (got %T)", ErrStdioUnavailable, name, requirement.Path, stream)
		}
		return file, nil
	}
	var flags int
	switch requirement.Mode {
	case FileWrite:
		flags = os.O_WRONLY
	case FileReadWrite:
		flags = os.O_RDWR
	default:
		flags = os.O_RDONLY
	}
	if requirement.Create {
		flags |= os.O_CREATE
	}
	if requirement.Append {
		flags |= os.O_APPEND
	}
	if requirement.Truncate {
		flags |= os.O_TRUNC
	}
	permission := requirement.Permission
	if permission == 0 {
		permission = 0600
	}
	var file *os.File
	var err error
	if requirement.AllowSpecial {
		file, err = os.OpenFile(requirement.Path, flags, permission)
	} else {
		// Do not truncate before validating the actual opened file type.
		file, err = openRegularCodecFile(requirement.Path, flags&^os.O_TRUNC, permission)
	}
	if file != nil {
		scope.OnClose(file.Close)
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !requirement.AllowSpecial {
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: %s (%s)", ErrSpecialFile, requirement.Path, info.Mode().Type())
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if requirement.Truncate {
			if err := file.Truncate(0); err != nil {
				return nil, err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return file, nil
}

func validateFileRequirement(requirement FileRequirement) error {
	if requirement.Path == "" {
		return fmt.Errorf("%w: file path is empty", ErrIllegalArgument)
	}
	if requirement.Mode > FileReadWrite {
		return fmt.Errorf("%w: invalid file mode", ErrIllegalArgument)
	}
	if requirement.Mode == FileRead && (requirement.Create || requirement.Append || requirement.Truncate) {
		return fmt.Errorf("%w: read-only file cannot create, append or truncate", ErrIllegalArgument)
	}
	if requirement.Append && requirement.Truncate {
		return fmt.Errorf("%w: append and truncate are mutually exclusive", ErrIllegalArgument)
	}
	if requirement.Permission&^fs.ModePerm != 0 {
		return fmt.Errorf("%w: file permission must contain only permission bits", ErrIllegalArgument)
	}
	if requirement.AllowStdio && isStdioPath(requirement.Path) {
		if requirement.Mode == FileReadWrite {
			return fmt.Errorf("%w: stdio alias %q does not support read/write access", ErrIllegalArgument, requirement.Path)
		}
		if requirement.Mode == FileRead && requirement.Path == "stdout" || requirement.Mode == FileWrite && requirement.Path == "stdin" {
			return fmt.Errorf("%w: stdio alias %q does not match file mode", ErrIllegalArgument, requirement.Path)
		}
		if requirement.Append || requirement.Truncate {
			return fmt.Errorf("%w: stdio alias %q cannot append or truncate", ErrIllegalArgument, requirement.Path)
		}
	}
	return nil
}

func isStdioPath(path string) bool {
	return path == "-" || path == "stdin" || path == "stdout"
}
