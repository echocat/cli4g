function __@FUNCTION@
    # Only the executable is taken from Fish's token output. The raw process and
    # cursor prefix preserve empty/multiline arguments and words after the cursor.
    set -l program
    for token in (commandline -opc)
        if contains -- "$token" command exec; or string match -qr '^[A-Za-z_][A-Za-z0-9_]*=' -- "$token"
            continue
        end
        set program "$token"
        break
    end
    test -n "$program"; or return
    set -l process (commandline -p | string collect --allow-empty --no-trim-newlines)
    set -l prefix (commandline -cp | string collect --allow-empty --no-trim-newlines)
    set -l response ("$program" @FISHREQUEST@ 1 fish "$prefix" "$process" 2>/dev/null | string split0)
    test (count $response) -ge 6; and test "$response[1]" = 1; or return
    set -l i 7
    while test $i -lt (count $response)
        set -l next (math $i + 1)
        set -l description (string replace --all --regex '[\t\r\n]' ' ' -- "$response[$next]" | string collect --allow-empty)
        printf '%s\t%s\0' "$response[$i]" "$description"
        set i (math $i + 2)
    end
    if test "$response[2]" != 0
        set -l search (string escape -- "$response[6]")
        for path in (complete -C "'' $search")
            set -l fields (string split --max 1 \t -- "$path")
            if test "$response[2]" = 2; and not string match -q '*/' -- "$fields[1]"
                continue
            end
            printf '%s%s\0' "$response[5]" "$path"
        end
    end
end
complete --command @FISHCOMMAND@ --erase
# The core already sorts unless KeepOrder is set. A single rule needs one query,
# with no persistent result cache or condition-dependent repeat invocations.
complete --command @FISHCOMMAND@ --no-files --keep-order --arguments '(__@FUNCTION@ | string split0)'
