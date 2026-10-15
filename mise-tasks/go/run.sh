#!/bin/sh
#MISE description="Build and run a temporary native Go binary"
#MISE hide=true
#MISE raw_args=true
#MISE quiet=true
#MISE raw=true
#MISE dir="{{config_root}}"

if [ "$#" -lt 2 ]; then
    printf '%s\n' 'usage: go:run <existing-working-directory> <package> [arguments...]' >&2
    exit 6
fi

working_directory=$1
package=$2
shift 2
case "$working_directory" in
    /*) ;;
    *) working_directory=./$working_directory ;;
esac
if [ ! -d "$working_directory" ]; then
    printf 'working directory does not exist: %s\n' "$working_directory" >&2
    exit 6
fi
if ! CDPATH='' cd -P "$working_directory"; then
    exit 6
fi

temporary=$(mktemp -d "${TMPDIR:-/tmp}/cli4g-go-run.XXXXXXXXXX") || exit 6
# Called indirectly by the EXIT trap below.
# shellcheck disable=SC2329
cleanup() {
    status=$?
    trap - EXIT
    rm -rf "$temporary"
    exit "$status"
}
trap 'cleanup' EXIT
temporary=$(CDPATH='' cd -P "$temporary" && pwd -P) || exit 6
# Wait for the foreground binary to handle Ctrl+C and return its own status.
trap ':' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

GOWORK=off
export GOWORK
GOOS=$(go env GOHOSTOS) || exit 6
GOARCH=$(go env GOHOSTARCH) || exit 6
export GOOS GOARCH
binary=$temporary/program
if ! go build -o "$binary" "$package"; then
    exit 6
fi
if [ ! -x "$binary" ]; then
    printf '%s\n' 'built program is not executable' >&2
    exit 6
fi

# A failed exec retains this trap; successful exec replaces the shell entirely.
# Thus start failures become 6, but real program exits 126/127 stay unchanged.
(
    trap 'exit 6' 0
    exec "$binary" "$@"
    exit 6
)
status=$?
exit "$status"
