param(
    [string]$Output = ".\\ojtools.exe"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-OJToolsIcon {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    Add-Type -AssemblyName System.Drawing

    $dir = Split-Path -Parent $Path
    if ($dir) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }

    $sourcePng = Join-Path $dir "material_compare_arrows_48.png"
    $iconUrl = "https://raw.githubusercontent.com/google/material-design-icons/master/android/action/compare_arrows/materialicons/black/res/drawable-xxxhdpi/baseline_compare_arrows_black_48.png"
    Invoke-WebRequest -Uri $iconUrl -OutFile $sourcePng

    $size = 64
    $bitmap = New-Object System.Drawing.Bitmap $size, $size
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $graphics.Clear([System.Drawing.Color]::Transparent)

    $pathFigure = New-Object System.Drawing.Drawing2D.GraphicsPath
    $corner = 16
    $rect = New-Object System.Drawing.Rectangle 4, 4, 56, 56
    $diameter = $corner * 2
    $pathFigure.AddArc($rect.X, $rect.Y, $diameter, $diameter, 180, 90)
    $pathFigure.AddArc($rect.Right - $diameter, $rect.Y, $diameter, $diameter, 270, 90)
    $pathFigure.AddArc($rect.Right - $diameter, $rect.Bottom - $diameter, $diameter, $diameter, 0, 90)
    $pathFigure.AddArc($rect.X, $rect.Bottom - $diameter, $diameter, $diameter, 90, 90)
    $pathFigure.CloseFigure()

    $backgroundBrush = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(255, 27, 48, 86))
    $graphics.FillPath($backgroundBrush, $pathFigure)

    $materialBitmap = [System.Drawing.Bitmap]::FromFile($sourcePng)
    try {
        $scaled = New-Object System.Drawing.Bitmap 34, 34
        $scaledGraphics = [System.Drawing.Graphics]::FromImage($scaled)
        try {
            $scaledGraphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
            $scaledGraphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $scaledGraphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
            $scaledGraphics.Clear([System.Drawing.Color]::Transparent)
            $scaledGraphics.DrawImage($materialBitmap, 0, 0, 34, 34)
        }
        finally {
            $scaledGraphics.Dispose()
        }

        for ($y = 0; $y -lt $scaled.Height; $y++) {
            for ($x = 0; $x -lt $scaled.Width; $x++) {
                $pixel = $scaled.GetPixel($x, $y)
                if ($pixel.A -gt 0) {
                    $scaled.SetPixel($x, $y, [System.Drawing.Color]::FromArgb($pixel.A, 255, 255, 255))
                }
            }
        }

        $graphics.DrawImage($scaled, 15, 15, 34, 34)
        $scaled.Dispose()
    }
    finally {
        $materialBitmap.Dispose()
        $backgroundBrush.Dispose()
        $pathFigure.Dispose()
        $graphics.Dispose()
    }

    $pngStream = New-Object System.IO.MemoryStream
    try {
        $bitmap.Save($pngStream, [System.Drawing.Imaging.ImageFormat]::Png)
        $pngBytes = $pngStream.ToArray()
    }
    finally {
        $bitmap.Dispose()
        $pngStream.Dispose()
    }

    $imageSize = $pngBytes.Length

    $stream = [System.IO.File]::Open($Path, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
    try {
        $writer = New-Object System.IO.BinaryWriter($stream)

        $writer.Write([UInt16]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]1)

        $writer.Write([byte]$size)
        $writer.Write([byte]$size)
        $writer.Write([byte]0)
        $writer.Write([byte]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]32)
        $writer.Write([UInt32]$imageSize)
        $writer.Write([UInt32]22)
        $writer.Write($pngBytes)
        $writer.Flush()
    }
    finally {
        $stream.Dispose()
    }
}

$root = $PSScriptRoot
Push-Location $root
try {
    $go = (Get-Command go -ErrorAction SilentlyContinue).Source
    if (-not $go) {
        throw "go executable was not found in PATH."
    }

    $gopath = (& $go env GOPATH).Trim()
    if (-not $gopath) {
        throw "GOPATH could not be resolved."
    }

    $rsrc = Join-Path $gopath "bin\\rsrc.exe"
    if (-not (Test-Path $rsrc)) {
        & $go install github.com/akavel/rsrc@latest
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path $rsrc)) {
            throw "rsrc tool installation failed."
        }
    }

    $buildDir = Join-Path $root ".build"
    $iconPath = Join-Path $buildDir "ojtools.ico"
    $sysoPath = Join-Path $root "cmd\\ojtools\\ojtools.syso"

    New-OJToolsIcon -Path $iconPath
    & $rsrc -ico $iconPath -o $sysoPath
    if ($LASTEXITCODE -ne 0) {
        throw "rsrc failed with exit code $LASTEXITCODE."
    }

    & $go build -ldflags "-H=windowsgui" -o $Output .\\cmd\\ojtools
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE."
    }

    Write-Host "Built GUI executable with icon at $Output"
}
finally {
    Pop-Location
}
