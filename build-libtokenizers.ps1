<#
.SYNOPSIS
  Builds libtokenizers.a (daulet/tokenizers, GNU target) for use as zolam's
  Windows native tokenizers library.

.DESCRIPTION
  daulet/tokenizers only publishes prebuilt libtokenizers.a for linux/darwin,
  not windows-amd64, so zolam's build fetches it from source on Windows. This
  script does that build standalone:
    - Rust: installed via Scoop, forced to the GNU host target, since Go's cgo
      linking in zolam expects the GNU/MinGW ABI, not MSVC.
    - GCC: installed via Scoop's standalone "mingw" package (a plain
      MinGW-w64 GCC build) — no MSYS2 needed. A real C compiler is required
      here, not just Rust's own linker: one of tokenizers' dependencies
      (onig_sys, wrapping Oniguruma) has a build.rs that compiles C source,
      and the GNU-target Rust toolchain needs a matching gcc to link.

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

Write-Step "Installing GCC via Scoop (mingw package)"
scoop bucket list | Select-String -Quiet '^main$' | Out-Null
if (-not $?) { scoop bucket add main }
if (-not (scoop list mingw 2>$null | Select-String "mingw")) {
    scoop install mingw
} else {
    Write-Host "  mingw already installed, skipping"
}

if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    throw "gcc still not on PATH after installing Scoop's mingw package. Open a new shell (PATH changes may need a fresh session) and re-run."
}
if (-not (Get-Command dlltool -ErrorAction SilentlyContinue)) {
    throw "dlltool still not on PATH after installing Scoop's mingw package. Open a new shell (PATH changes may need a fresh session) and re-run."
}

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
