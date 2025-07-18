Get-ChildItem -Recurse -Filter *.go | ForEach-Object {
    $path = $_.FullName
    $content = Get-Content $path

    $usesRepoChat = $content -match 'repoChat\.'
    $usesModelChat = $content -match 'modelChat\.'

    if ($usesRepoChat -or $usesModelChat) {
        # Найти строку package
        $insertIndex = 0
        for ($i = 0; $i -lt $content.Count; $i++) {
            if ($content[$i] -match '^package\s') {
                $insertIndex = $i + 1
                break
            }
        }

        $imports = @()

        # Пропустить пустую строку после package, если она уже есть
        if ($content[$insertIndex] -match '^\s*$') {
            $insertIndex += 1
        } else {
            $imports += ""  # Вставить пустую строку после package
        }

        $imports += "import ("
        if ($usesRepoChat) {
            $imports += '    repoChat "mwork_backend/internal/repositories/chat"'
        }
        if ($usesModelChat) {
            $imports += '    modelChat "mwork_backend/internal/models/chat_model"'
        }
        $imports += ")"
        $imports += ""  # пустая строка после блока import

        $before = $content[0..($insertIndex - 1)]
        $after = $content[$insertIndex..($content.Count - 1)]
        $newContent = $before + $imports + $after

        Set-Content -Path $path -Value $newContent
        Write-Host "✅ Fixed imports in $path"
    }
}
