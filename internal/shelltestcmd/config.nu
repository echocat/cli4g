$env.config.show_banner = false
$env.config.edit_mode = "emacs"
$env.config.completions.quick = true
$env.config.completions.partial = true
$env.PROMPT_COMMAND = {|| "E2E> "}
$env.PROMPT_COMMAND_RIGHT = ""
$env.PROMPT_INDICATOR = ""
$env.config.keybindings ++= [{
    name: e2e_dump
    modifier: control
    keycode: char_o
    mode: [emacs]
    event: {send: executehostcommand, cmd: 'print ("\n__CLI4G_BUFFER__" + (commandline | to json --raw) + "__CLI4G_END__")'}
}]
