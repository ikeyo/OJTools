param(
    [ValidateSet("enable", "disable", "status")]
    [string]$Action = "status",
    [string]$Name = "OJTools",
    [string]$ExePath = (Join-Path $PSScriptRoot "OJTools.exe")
)

$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$command = '"' + $ExePath + '" run'

switch ($Action) {
    "enable" {
        if (-not (Test-Path $ExePath)) {
            throw "Executable not found: $ExePath"
        }

        New-Item -Path $runKey -Force | Out-Null
        New-ItemProperty -Path $runKey -Name $Name -PropertyType String -Value $command -Force | Out-Null
        Write-Host "Enabled auto-start:"
        Write-Host $command
    }
    "disable" {
        Remove-ItemProperty -Path $runKey -Name $Name -ErrorAction SilentlyContinue
        Write-Host "Disabled auto-start for $Name"
    }
    "status" {
        $current = (Get-ItemProperty -Path $runKey -ErrorAction SilentlyContinue).$Name
        if ($current) {
            Write-Host "Auto-start is enabled:"
            Write-Host $current
        } else {
            Write-Host "Auto-start is not enabled."
        }
    }
}
