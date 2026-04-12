param(
    [string]$Output = ".\OJTools.exe"
)

$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) {
    throw "go executable was not found in PATH."
}

& $go build -ldflags "-H=windowsgui" -o $Output .\cmd\ojtools
if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE."
}

Write-Host "Built GUI executable at $Output"
