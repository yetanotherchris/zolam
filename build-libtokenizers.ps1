<#
.SYNOPSIS
  Builds libtokenizers.a (daulet/tokenizers, GNU target) for use as zolam's
  Windows native tokenizers library.

.DESCRIPTION
  daulet/tokenizers only publishes prebuilt libtokenizers.a for linux/darwin,
  not windows-amd64, so zolam's build fetches it from source on Windows. This
  script does that build standalone, matching CI's approach (see
  .github/workflows/build-release.yml's "Build libtokenizers.a from source
  (Windows)" step):
    - Rust: installed via Scoop, forced to the GNU host target, since Go's cgo
      linking in zolam expects the GNU/MinGW ABI, not MSVC.
    - GCC: installed via MSYS2/pacman — cargo needs a matching C linker for the
      GNU target.

  Assumes Git is already available on PATH (this script won't install it).

.PARAMETER TokenizersVersion
  daulet/tokenizers tag to build. Defaults to the version currently pinned in
  zolam's go.mod (check src/go.mod's `require github.com/daulet/tokenizers`
  line if this drifts).

.PARAMETER OutputPath
  Where to copy the built libtokenizers.a. Defaults to .\libtokenizers.a in the
  current directory. Copy it into a zolam checkout at
  src\native\tokenizers\libtokenizers.a before building zolam itself.

.EXAMPLE
  .\build-libtokenizers.ps1

.EXAMPLE
  .\build-libtokenizers.ps1 -TokenizersVersion v1.27.0 -OutputPath C:\src\zolam\src\native\tokenizers\libtokenizers.a
#>
[CmdletBinding()]
param(
    [string]$TokenizersVersion = "v1.27.0",
    [string]$OutputPath = (Join-Path (Get-Location) "libtokenizers.a")
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
    Write-Host ""
    Write-Host "==> $msg" -ForegroundColor Cyan
}

function Require-Scoop {
    if (-not (Get-Command scoop -ErrorAction SilentlyContinue)) {
        throw "Scoop isn't installed. Install it first: https://scoop.sh"
    }
}

# ---------------------------------------------------------------------------
Write-Step "Checking Scoop"
Require-Scoop

Write-Step "Installing Rust via Scoop"
if (-not (scoop list rustup 2>$null | Select-String "rustup")) {
    scoop install rustup
} else {
    Write-Host "  rustup already installed, skipping"
}

Write-Step "Forcing Rust to the GNU host target (zolam's cgo linking expects MinGW ABI, not MSVC)"
rustup toolchain install stable-x86_64-pc-windows-gnu
rustup default stable-x86_64-pc-windows-gnu

Write-Step "Checking for MSYS2 (needed for a matching GCC to link the Rust staticlib)"
$msys2Bash = "C:\msys64\usr\bin\bash.exe"
if (-not (Test-Path $msys2Bash)) {
    if (-not (scoop list msys2 2>$null | Select-String "msys2")) {
        scoop bucket list | Select-String -Quiet '^extras$' | Out-Null
        if (-not $?) { scoop bucket add extras }
        scoop install extras/msys2
    }
    $msys2Bash = (Get-Command msys2 -ErrorAction SilentlyContinue)?.Source
    if (-not $msys2Bash) {
        throw "Could not locate msys2 after installing via Scoop. Install MSYS2 manually from https://www.msys2.org and re-run."
    }
}
$msys2Root = Split-Path (Split-Path (Split-Path $msys2Bash))
$mingw64 = Join-Path $msys2Root "mingw64"

Write-Step "Installing GCC via pacman (matches CI's toolchain)"
& $msys2Bash -lc "pacman -Syu --noconfirm"
& $msys2Bash -lc "pacman -S --noconfirm --needed mingw-w64-x86_64-gcc"

# Put MSYS2's mingw64 toolchain on PATH for this session so `gcc`/`g++`
# resolve to it for the cargo build below.
$env:Path = "$mingw64\bin;$env:Path"

Write-Step "Cloning daulet/tokenizers ($TokenizersVersion)"
$tokSrc = Join-Path $env:TEMP "tokenizers-src"
if (Test-Path $tokSrc) { Remove-Item -Recurse -Force $tokSrc }
git clone --depth 1 --branch $TokenizersVersion https://github.com/daulet/tokenizers.git $tokSrc

Write-Step "Building tokenizers-ffi (release, x86_64-pc-windows-gnu)"
cargo build --release --manifest-path "$tokSrc\Cargo.toml" -p tokenizers-ffi --target x86_64-pc-windows-gnu

Write-Step "Copying libtokenizers.a to $OutputPath"
New-Item -ItemType Directory -Force -Path (Split-Path $OutputPath) | Out-Null
Copy-Item "$tokSrc\target\x86_64-pc-windows-gnu\release\libtokenizers_ffi.a" $OutputPath -Force

Write-Host ""
Write-Host "Built: $OutputPath" -ForegroundColor Green
Write-Host "Copy this into a zolam checkout at src\native\tokenizers\libtokenizers.a" -ForegroundColor Yellow
Write-Host "before building zolam.exe (fetchnative will see it and skip its own download)." -ForegroundColor Yellow
