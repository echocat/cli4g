package cli

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPowerShellProfileInstallationPreservesContentAndEncoding(t *testing.T) {
	for _, encoding := range []string{"utf8", "utf8-bom", "utf16-le", "utf16-be", "ansi"} {
		t.Run(encoding, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "profile.ps1")
			script := filepath.Join(directory, "über's completions.ps1")
			original := []byte("# user's profile: über\r\n$env:KEEP = 'yes'\r\n")
			switch encoding {
			case "utf8-bom":
				original = append([]byte{0xef, 0xbb, 0xbf}, original...)
			case "utf16-le", "utf16-be":
				var order binary.ByteOrder = binary.LittleEndian
				if encoding == "utf16-be" {
					order = binary.BigEndian
				}
				units := utf16.Encode([]rune(string(original)))
				original = make([]byte, (len(units)+1)*2)
				order.PutUint16(original, 0xfeff)
				for i, unit := range units {
					order.PutUint16(original[(i+1)*2:], unit)
				}
			case "ansi":
				original = []byte("# user's profile: \xfcber\r\n$env:KEEP = 'yes'\r\n")
			}
			require.NoError(t, os.WriteFile(path, original, 0600))
			require.NoError(t, installPowerShellProfile(context.Background(), path, script, "ship"))
			installed, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, original, installed[:len(original)])
			require.Greater(t, len(installed), len(original))
			require.NoError(t, installPowerShellProfile(context.Background(), path, script, "ship"))
			again, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, installed, again)
			require.ErrorContains(t, installPowerShellProfile(context.Background(), path, script+"other", "ship"), "different cli4g loader")
		})
	}
}
