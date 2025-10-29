# make-structure.ps1
# Создаёт дерево структуры проекта, игнорируя node_modules, .git, dist и т.п.

function Show-Tree($path, $indent = "") {
    Get-ChildItem $path | Where-Object {
        $_.Name -notmatch '^(node_modules|\.git|dist|build|\.next)$'
    } | ForEach-Object {
        Write-Output "$indent|-- $($_.Name)"
        if ($_.PSIsContainer) {
            Show-Tree $_.FullName ("$indent|   ")
        }
    }
}

# Основная часть
$OutputFile = "structure.txt"
Show-Tree . | Out-File $OutputFile -Encoding utf8
Write-Host "`n✅ Структура проекта сохранена в $OutputFile"
