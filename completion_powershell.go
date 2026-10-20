package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
)

//go:embed completion_powershell.ps1
var powershellCompletionScript string

func quotePowerShell(value string) string {
	return "'" + strings.NewReplacer("'", "''", "\u2018", "\u2018\u2018", "\u2019", "\u2019\u2019", "\u201a", "\u201a\u201a", "\u201b", "\u201b\u201b").Replace(value) + "'"
}

func powershellCompletionProfiles(ctx context.Context) ([]string, error) {
	var profiles []string
	seen := make(map[string]bool)
	for _, name := range []string{"pwsh", "powershell"} {
		shell, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		output, err := exec.CommandContext(ctx, shell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); [Console]::Write($PROFILE.CurrentUserAllHosts)").Output()
		if err != nil {
			return nil, fmt.Errorf("could not discover %s user profile: %w", name, err)
		}
		path := strings.TrimSpace(strings.TrimPrefix(string(output), "\ufeff"))
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("%s returned no absolute user profile path", name)
		}
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if !seen[key] {
			profiles = append(profiles, path)
			seen[key] = true
		}
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("could not discover PowerShell user profiles: install pwsh or powershell, or supply an installation directory")
	}
	return profiles, nil
}

func installPowerShellProfile(ctx context.Context, profile, script, command string) error {
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
	hash := sha256.Sum256([]byte(command))
	marker := fmt.Sprintf("# cli4g completion %x", hash[:8])
	// ASCII source preserves legacy ANSI profiles too. The path is data, never
	// interpolated PowerShell code, including when it contains quotes or Unicode.
	block := "\r\n" + marker + "\r\n. ([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String('" +
		base64.StdEncoding.EncodeToString([]byte(script)) + "')))\r\n# end cli4g completion\r\n"
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
		return fmt.Errorf("PowerShell profile already contains a different cli4g loader for %q: %s", command, profile)
	}
	return replaceCompletionFile(ctx, profile, append(data, addition...), mode)
}
