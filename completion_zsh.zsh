_@FUNCTION@() {
    emulate -L zsh
    setopt extendedglob
    local program="${(Q)words[1]}" field mode value description i
    local -a response values displays addopts args
    args=("${(@Q)words[2,-1]}")
    # Zsh's cursor fragments already account for an unfinished quote, unlike
    # dequoting the full raw word, which requires a matching closing quote.
    args[CURRENT-1]="${(Q)PREFIX}${(Q)SUFFIX}"
    while IFS= read -r -d '' field; do
        response+=("$field")
    done < <("$program" @BASHREQUEST@ 1 zsh "$((CURRENT - 2))" "${(Q)PREFIX}" "${args[@]}" 2>/dev/null)
    (( ${#response} >= 6 )) && [[ $response[1] == 1 ]] || return 0
    if [[ $response[4] == 1 ]]; then
        addopts=(-V @FUNCTION@)
    else
        addopts=(-J @FUNCTION@)
    fi
    if [[ $response[3] == 1 ]] || (( CURRENT < ${#words} )); then
        addopts+=(-S '')
    fi
    for ((i=7; i<${#response}; i+=2)); do
        value=$response[i]
        description=$response[i+1]
        values+=("$value")
        displays+=("$value${description:+ -- $description}")
    done
    (( ${#values} )) && compadd -U "${addopts[@]}" -d displays -- "${values[@]}"
    mode=$response[2]
    if [[ $mode != 0 ]]; then
        [[ -n $response[5] ]] && compset -P "${(b)response[5]}"
        if [[ $mode == 2 ]]; then
            _files -/
        else
            _files
        fi
    fi
    return 0
}
if [[ $funcstack[1] == @ZSHAUTOLOAD@ ]]; then
    _@FUNCTION@ "$@"
else
    compdef _@FUNCTION@ @BASHCOMMAND@
fi
