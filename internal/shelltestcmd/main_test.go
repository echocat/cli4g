package main

import (
	"fmt"
	"testing"
)

func TestChooseEngine(t *testing.T) {
	for _, test := range []struct {
		value     string
		available map[string]string
		want      string
	}{
		{"auto", map[string]string{"podman": "/bin/podman", "docker": "/bin/docker"}, "/bin/podman"},
		{"auto", map[string]string{"docker": "/bin/docker"}, "/bin/docker"},
		{"docker", map[string]string{"podman": "/bin/podman", "docker": "/bin/docker"}, "/bin/docker"},
		{"podman", map[string]string{"docker": "/bin/docker"}, ""},
		{"auto", nil, ""},
		{"unknown", map[string]string{"unknown": "/bin/unknown"}, ""},
	} {
		got, err := chooseEngine(test.value, func(name string) (string, error) {
			if path := test.available[name]; path != "" {
				return path, nil
			}
			return "", fmt.Errorf("not installed")
		})
		if got != test.want || (err != nil) != (test.want == "") {
			t.Fatalf("chooseEngine(%q): %q, %v; want %q", test.value, got, err, test.want)
		}
	}
}
