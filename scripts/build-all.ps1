param([string]$Version = "dev")
$ErrorActionPreference = "Stop"
$project = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $project "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

$targets = @(
    @{ OS = "windows"; Arch = "amd64"; Suffix = ".exe" },
    @{ OS = "windows"; Arch = "arm64"; Suffix = ".exe" },
    @{ OS = "linux"; Arch = "amd64"; Suffix = "" },
    @{ OS = "linux"; Arch = "arm64"; Suffix = "" },
    @{ OS = "linux"; Arch = "arm"; Arm = "7"; Suffix = "" },
    @{ OS = "darwin"; Arch = "amd64"; Suffix = "" },
    @{ OS = "darwin"; Arch = "arm64"; Suffix = "" }
)

Push-Location $project
try {
    foreach ($target in $targets) {
        $env:CGO_ENABLED = "0"
        $env:GOOS = $target.OS
        $env:GOARCH = $target.Arch
        if ($target.Arm) { $env:GOARM = $target.Arm } else { Remove-Item Env:GOARM -ErrorAction SilentlyContinue }
        $label = "$($target.OS)-$($target.Arch)"
        if ($target.Arm) { $label += "v$($target.Arm)" }
        $output = Join-Path $dist "shtu-net-login-$label$($target.Suffix)"
        Write-Host "Building $label"
        go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $output ./cmd/shtu-net-login
    }
} finally {
    Pop-Location
    Remove-Item Env:GOOS,Env:GOARCH,Env:GOARM,Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
