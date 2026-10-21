_@FUNCTION@_quote_context() {
    # COMP_POINT is a byte offset, independently of the interactive locale.
    # Only inspect syntax; never evaluate the command line or provider values.
    local LC_ALL=C line="${COMP_LINE:-}" char next i
    quote=
    for ((i=0; i<${COMP_POINT:-0}; i++)); do
        char="${line:i:1}"
        next="${line:i+1:1}"
        if [[ "$char" == \\ && "$quote" != "'" ]] &&
            [[ -z "$quote" || "$quote" == ansi || "$next" == \\ || "$next" == '"' ||
               "$next" == '$' || "$next" == '`' || "$next" == $'\n' ]]; then
            ((i++))
        elif [[ -n "$quote" && ( "$char" == "$quote" || ( "$quote" == ansi && "$char" == "'" ) ) ]]; then
            quote=
        elif [[ -z "$quote" && "$char" == '$' && "$next" == "'" ]]; then
            quote=ansi
            ((i++))
        elif [[ -z "$quote" && ( "$char" == "'" || "$char" == '"' ) ]]; then
            quote="$char"
        fi
    done
}

_@FUNCTION@() {
    local command="$1" field mode path candidate quote i
    local -a response=()
    while IFS= read -r -d '' field; do
        response+=("$field")
    done < <("$command" @BASHREQUEST@ 1 bash "${COMP_POINT:-0}" "${COMP_LINE:-}" "${2:-}" 2>/dev/null)
    COMPREPLY=()
    ((${#response[@]} >= 7)) && [[ "${response[0]}" == 1 ]] || return 0
    [[ "${response[2]}" == 1 ]] && compopt -o nospace
    [[ "${response[3]}" == 1 ]] && compopt -o nosort
    for ((i=7; i<${#response[@]}; i++)); do
        COMPREPLY+=("${response[i]}")
    done
    mode="${response[1]}"
    if [[ "$mode" != 0 ]]; then
        local action=-f
        [[ "$mode" == 2 ]] && action=-d
        while IFS= read -r path; do
            candidate="${response[4]}$path"
            candidate="${candidate#"${response[6]}"}"
            if [[ -d "$path" ]]; then
                candidate="${candidate%/}/"
                compopt -o nospace
            fi
            COMPREPLY+=("$candidate")
        done < <(compgen "$action" -- "${response[5]}")
    fi
    # Readline's filename quoting can leave leading $ and backticks active.
    # Supply literal shell syntax ourselves, preserving an unfinished quote.
    _@FUNCTION@_quote_context
    # The request lexer supports simple quotes, not ANSI-C escape decoding.
    # Do not insert single-quote syntax inside an active $'...' expression.
    if [[ "$quote" == ansi ]]; then
        COMPREPLY=()
        return 0
    fi
    for i in "${!COMPREPLY[@]}"; do
        candidate="${COMPREPLY[i]}"
        case "$quote" in
            "'") candidate="'${candidate//\'/\'\\\'\'}'" ;;
            '"')
                candidate="${candidate//\\/\\\\}"
                candidate="${candidate//\$/\\\$}"
                candidate="${candidate//\`/\\\`}"
                candidate="${candidate//\"/\\\"}"
                candidate="${candidate//\!/\"\'\!\'\"}"
                candidate="\"$candidate\""
                ;;
            *) printf -v candidate '%q' "$candidate" ;;
        esac
        COMPREPLY[i]="$candidate"
    done
    return 0
}
complete -o noquote -F _@FUNCTION@ -- @BASHCOMMAND@
