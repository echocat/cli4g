PS1='E2E> '
source /usr/share/bash-completion/bash_completion
bind 'set enable-bracketed-paste off'
bind 'set show-all-if-ambiguous off'
bind '"\C-x\C-n": menu-complete'
_e2e_dump() {
    local value="$READLINE_LINE"
    printf '\n__CLI4G_BUFFER__'
    printf '%s' "$value" | cli4g-completion-test __quote-stdin
    printf '__CLI4G_END__\n'
}
bind -x '"\C-o": _e2e_dump'
printf '\n__CLI4G_READY__\n'
