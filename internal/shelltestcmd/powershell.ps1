Import-Module PSReadLine
$OutputEncoding = [System.Text.Encoding]::ASCII
Set-PSReadLineOption -EditMode Emacs -PredictionSource None -BellStyle None
Set-PSReadLineKeyHandler -Chord Tab -Function TabCompleteNext
Set-PSReadLineKeyHandler -Chord Ctrl+o -ScriptBlock {
    $line = ''
    $cursor = 0
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    [Console]::WriteLine()
    [Console]::WriteLine('__CLI4G_BUFFER__' + (ConvertTo-Json -InputObject $line -Compress) + '__CLI4G_END__')
    [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, '')
    [Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine()
}
function global:prompt { 'E2E> ' }
[Console]::WriteLine('__CLI4G_READY__')
