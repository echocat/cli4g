Register-ArgumentCompleter -CommandName @(@PSCOMMAND@, @PSEXE@) -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    # All helpers and preferences belong to this invocation, not the user session.
    Set-StrictMode -Off
    $ErrorActionPreference = 'Stop'
    function Read-Literal($element) {
        if ($element -is [System.Management.Automation.Language.StringConstantExpressionAst] -or
            ($element -is [System.Management.Automation.Language.ExpandableStringExpressionAst] -and
             $element.NestedExpressions.Count -eq 0)) {
            return [string]$element.Value
        }
        # Decode only our fixed literal wrapper, never evaluate arbitrary ASTs.
        if ($element -is [System.Management.Automation.Language.ArrayExpressionAst] -and
            $element.Extent.Text -ceq "@('--%')") {
            return '--%'
        }
        return [string]$element.Extent.Text
    }
    function Read-Prefix([string]$text) {
        $tokens = $null
        $errors = $null
        $ast = [System.Management.Automation.Language.Parser]::ParseInput('x ' + $text, [ref]$tokens, [ref]$errors)
        $elements = $ast.EndBlock.Statements[0].PipelineElements[0].CommandElements
        if ($elements.Count -eq 2) { return Read-Literal $elements[1] }
        return $text
    }
    function Quote-Literal([string]$value) {
        # The native binder removes even a quoted scalar --% and changes how
        # later arguments are passed. A fixed singleton array avoids that path.
        if ($value -ceq '--%') { return "@('--%')" }
        if ($value -match '^[-\p{L}\p{N}_./:\\=+]+$') { return $value }
        # Legacy native argument passing strips unescaped embedded double quotes.
        if ($PSVersionTable.PSVersion -lt [version]'7.3' -or $PSNativeCommandArgumentPassing -eq 'Legacy') {
            $value = [regex]::Replace($value, '(\\*)"', '$1$1\"')
        }
        return "'" + ($value -replace "['\u2018\u2019\u201a\u201b]", '$&$&') + "'"
    }

    try {
        $words = [System.Collections.Generic.List[string]]::new()
        $word = -1
        $offset = 0
        foreach ($element in ($commandAst.CommandElements | Select-Object -Skip 1)) {
            $start = $element.Extent.StartOffset
            $end = $element.Extent.EndOffset
            if ($word -lt 0 -and $cursorPosition -lt $start) {
                $word = $words.Count
                $words.Add('')
            }
            if ($word -lt 0 -and $cursorPosition -ge $start -and $cursorPosition -le $end) {
                $word = $words.Count
                $prefix = Read-Prefix $element.Extent.Text.Substring(0, $cursorPosition - $start)
                $offset = [System.Text.Encoding]::UTF8.GetByteCount([string]$prefix)
            }
            $words.Add((Read-Literal $element))
        }
        if ($word -lt 0) {
            $word = $words.Count
            $words.Add('')
        }
        $request = @{words = $words.ToArray(); word = $word; offset = $offset} | ConvertTo-Json -Depth 4 -Compress
        $inputText = [System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($request))
        $program = $commandAst.GetCommandName()
        $response = $inputText | & $program @PSREQUEST@ 1 powershell 2>$null
        if ($LASTEXITCODE -ne 0) { return '' }
        $json = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String(($response -join '')))
        $result = $json | ConvertFrom-Json
        $emitted = 0
        foreach ($candidate in $result.candidates) {
            $description = [string]$candidate.description
            if (!$description) { $description = ' ' }
            [System.Management.Automation.CompletionResult]::new(
                (Quote-Literal $candidate.value), $candidate.value, 'ParameterValue', $description)
            $emitted++
        }
        if ($result.files -or $result.directories) {
            $search = [string]$result.fileInput
            # PowerShell's provider completer treats a colon as a drive even on
            # Unix. Native applications can also consume literal colon filenames.
            if ([System.IO.Path]::DirectorySeparatorChar -eq '/' -and $search -match '^([^/\\:]+):') {
                $directory = [System.IO.Path]::GetDirectoryName($search)
                if (!$directory) { $directory = '.' }
                $prefix = [System.IO.Path]::GetFileName($search)
                $paths = foreach ($entry in [System.IO.Directory]::EnumerateFileSystemEntries($directory)) {
                    $leaf = [System.IO.Path]::GetFileName($entry)
                    if (!$leaf.StartsWith($prefix, [System.StringComparison]::Ordinal)) { continue }
                    $kind = 'ProviderItem'
                    if ([System.IO.Directory]::Exists($entry)) {
                        $kind = 'ProviderContainer'
                        $entry += '/'
                    }
                    [System.Management.Automation.CompletionResult]::new((Quote-Literal $entry), $leaf, $kind, $entry)
                }
            } else {
                $paths = [System.Management.Automation.CompletionCompleters]::CompleteFilename($search)
            }
            foreach ($path in $paths) {
                if ($result.directories -and $path.ResultType -ne 'ProviderContainer') { continue }
                $value = [string]$result.filePrefix + (Read-Prefix $path.CompletionText)
                [System.Management.Automation.CompletionResult]::new(
                    (Quote-Literal $value), $path.ListItemText, $path.ResultType, $path.ToolTip)
                $emitted++
            }
        }
        # An empty string suppresses PowerShell's implicit filename fallback.
        if ($emitted -eq 0) { return '' }
    } catch {
        return ''
    }
}
