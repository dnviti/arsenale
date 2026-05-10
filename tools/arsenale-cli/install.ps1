param(
    [string]$Version = $env:ARSENALE_VERSION,
    [string]$InstallDir = $env:ARSENALE_INSTALL_DIR,
    [string]$Repo = $env:ARSENALE_REPO
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Repo)) {
    $Repo = "dnviti/arsenale"
}

function Resolve-Arch {
    $processorArch = $env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrWhiteSpace($processorArch)) {
        $processorArch = $env:PROCESSOR_ARCHITECTURE
    }
    if ([string]::IsNullOrWhiteSpace($processorArch)) {
        throw "could not detect CPU architecture"
    }

    switch ($processorArch.ToUpperInvariant()) {
        "AMD64" { "amd64"; break }
        "ARM64" { "arm64"; break }
        default { throw "unsupported CPU architecture: $processorArch" }
    }
}

function Resolve-Version {
    param([string]$RequestedVersion, [string]$Repository)

    if ([string]::IsNullOrWhiteSpace($RequestedVersion) -or $RequestedVersion -eq "latest") {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repository/releases/latest" -Headers @{ "User-Agent" = "arsenale-cli-installer" }
        return ($release.tag_name -replace '^v', '')
    }

    return ($RequestedVersion -replace '^v', '')
}

function Get-DefaultInstallDir {
    if ($env:LOCALAPPDATA) {
        return (Join-Path $env:LOCALAPPDATA "Programs\Arsenale\bin")
    }
    return (Join-Path $HOME ".arsenale\bin")
}

function Add-UserPath {
    param([string]$Directory)

    if ($env:ARSENALE_SKIP_PATH -eq "1") {
        return
    }

    $currentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @()
    if (-not [string]::IsNullOrWhiteSpace($currentUserPath)) {
        $pathEntries = $currentUserPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }
    if ($pathEntries -contains $Directory) {
        return
    }

    $nextPath = if ([string]::IsNullOrWhiteSpace($currentUserPath)) { $Directory } else { "$currentUserPath;$Directory" }
    [Environment]::SetEnvironmentVariable("Path", $nextPath, "User")
    if (($env:Path -split ';') -notcontains $Directory) {
        $env:Path = "$env:Path;$Directory"
    }
    Write-Host "Added $Directory to the user PATH. Open a new terminal if arsenale is not found."
}

$resolvedVersion = Resolve-Version -RequestedVersion $Version -Repository $Repo
$arch = Resolve-Arch
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Get-DefaultInstallDir
}

$archiveName = "arsenale-cli_${resolvedVersion}_windows_${arch}.zip"
$downloadBase = "https://github.com/$Repo/releases/download/v$resolvedVersion"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("arsenale-cli-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $archivePath = Join-Path $tempDir $archiveName
    $checksumsPath = Join-Path $tempDir "checksums_sha256.txt"
    Write-Host "Installing Arsenale CLI $resolvedVersion for windows/$arch..."
    Invoke-WebRequest -Uri "$downloadBase/$archiveName" -OutFile $archivePath -Headers @{ "User-Agent" = "arsenale-cli-installer" }
    Invoke-WebRequest -Uri "$downloadBase/checksums_sha256.txt" -OutFile $checksumsPath -Headers @{ "User-Agent" = "arsenale-cli-installer" }

    $checksumLine = Get-Content $checksumsPath | Where-Object { $_ -match "\s+$([regex]::Escape($archiveName))$" } | Select-Object -First 1
    if (-not $checksumLine) {
        throw "checksum for $archiveName not found"
    }
    $expected = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "checksum mismatch for $archiveName"
    }

    Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $tempDir "arsenale.exe") -Destination (Join-Path $InstallDir "arsenale.exe") -Force
    Add-UserPath -Directory $InstallDir
    Write-Host "Installed: $(Join-Path $InstallDir 'arsenale.exe')"
} finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
