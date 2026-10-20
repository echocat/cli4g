package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf16"
)

// appendCompletionProfile retains user content, encoding, permissions and symlink
// targets. A different block under the same marker is never silently replaced.
func appendCompletionProfile(ctx context.Context, profile, marker, block string) error {
	if info, err := os.Lstat(profile); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(profile)
		if err != nil {
			return err
		}
		profile = resolved
	}
	data, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	mode := os.FileMode(0600)
	if err == nil {
		info, err := os.Stat(profile)
		if err != nil {
			return err
		}
		mode = info.Mode().Perm()
	}
	encode := func(value string) []byte { return []byte(value) }
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		var order binary.ByteOrder = binary.LittleEndian
		if data[0] == 0xfe {
			order = binary.BigEndian
		}
		encode = func(value string) []byte {
			units := utf16.Encode([]rune(value))
			result := make([]byte, len(units)*2)
			for i, unit := range units {
				order.PutUint16(result[i*2:], unit)
			}
			return result
		}
	}
	addition := encode(block)
	if bytes.Contains(data, addition) {
		return nil
	}
	if bytes.Contains(data, encode(marker)) {
		return fmt.Errorf("shell startup file already contains a different cli4g loader: %s", profile)
	}
	return replaceCompletionFile(ctx, profile, append(data, addition...), mode)
}
