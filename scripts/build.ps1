param([string]$Version = "dev")
$ErrorActionPreference = 'Stop'
$buildRoot = Split-Path -Parent $PSScriptRoot
$oldValues = @{}
foreach ($name in @('CGO_ENABLED', 'GOOS', 'GOARCH', 'GOAMD64')) {
    $oldValues[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
Push-Location $buildRoot
try {
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:GOAMD64 = 'v1'
    New-Item -ItemType Directory -Path dist -Force | Out-Null
    go build -mod=readonly -trimpath -ldflags "-s -w -X main.version=$Version" -o dist/panaino-bot .
    if ($LASTEXITCODE -ne 0) { throw 'Linux build failed' }
    $binaryHash = (Get-FileHash dist/panaino-bot -Algorithm SHA256).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText((Join-Path $buildRoot 'dist/SHA256SUMS'), "$binaryHash  panaino-bot`n", (New-Object System.Text.UTF8Encoding($false)))
    Write-Output "$binaryHash  panaino-bot"
} finally {
    Pop-Location
    foreach ($name in $oldValues.Keys) {
        [Environment]::SetEnvironmentVariable($name, $oldValues[$name], 'Process')
    }
}
