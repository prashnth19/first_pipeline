# unpack-archives.ps1
param (
    [string]$RootPath = "."
)

# Ensure 7-Zip is installed and in PATH
function Test-7Zip {
    $7z = Get-Command "7z" -ErrorAction SilentlyContinue
    if (-not $7z) {
        Write-Error "❌ 7-Zip is not installed or not in PATH. Please install it and try again."
        exit 1
    }
}

Test-7Zip

# Supported extensions
$extensions = @("*.zip", "*.jar", "*.rar", "*.tar", "*.tar.gz", "*.tgz")

foreach ($ext in $extensions) {
    Get-ChildItem -Path $RootPath -Recurse -Include $ext | ForEach-Object {
        $archive = $_.FullName
        $outputDir = Join-Path $_.DirectoryName ($_.BaseName + "_extracted")

        if (-not (Test-Path $outputDir)) {
            New-Item -ItemType Directory -Path $outputDir | Out-Null
        }

        Write-Host "📦 Extracting: $archive -> $outputDir"
        Start-Process -Wait -NoNewWindow -FilePath "7z" -ArgumentList "x `"$archive`" -o`"$outputDir`" -y"
    }
}

Write-Host "`n✅ Done extracting all supported archives."
