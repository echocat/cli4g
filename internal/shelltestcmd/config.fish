set -g fish_greeting
set -g fish_autosuggestion_enabled 0
function fish_prompt
    printf 'E2E> '
end
function fish_right_prompt
end
fish_default_key_bindings
function __e2e_dump
    printf '\n__CLI4G_BUFFER__'
    commandline --current-buffer | cli4g-completion-test __quote-stdin --trim-newline
    printf '__CLI4G_END__\n'
    commandline --replace ''
    commandline --function execute
end
bind \co __e2e_dump
printf '__CLI4G_READY__\n'
