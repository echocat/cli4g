PS1='E2E> '
setopt completeinword
bindkey -e
bindkey '^I' expand-or-complete
bindkey '^X^N' menu-complete
_e2e_dump() {
    print -rn -- $'\n__CLI4G_BUFFER__'
    print -rn -- "$BUFFER" | cli4g-completion-test __quote-stdin
    print -r -- '__CLI4G_END__'
    BUFFER=''
    zle accept-line
}
zle -N _e2e_dump
bindkey '^O' _e2e_dump
print -r -- '__CLI4G_READY__'
