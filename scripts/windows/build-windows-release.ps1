<#
.SYNOPSIS
  Builds zolam.exe on Windows and drops it at artifacts/zolam-windows-amd64.exe,
  for manual inclusion in a GitHub release (see build-release.yml's release job).

.DESCRIPTION
  CI's MSYS2-toolchain build currently can't link this binary — go-fitz's bundled
  MuPDF static lib and DuckDB's/the Rust tokenizers lib's Windows static libs pull
  in undefined symbols that the MSYS2 MinGW GCC build can't resolve (see the
  comment above the matrix in .github/workflows/build-release.yml). This script
  reproduces the same build locally so you can build/debug it directly instead of
  round-tripping through CI.

  Toolchain sourcing:
    - Rust: installed via Scoop (as requested), forced to the GNU host target,
      since Go's cgo linking here expects the GNU/MinGW ABI, not MSVC.
    - GCC + Tesseract + Leptonica: installed via MSYS2/pacman, matching CI. These
      need to come from one consistent toolchain — mixing MSYS2's GCC with a
      separately-sourced one (e.g. Scoop's own "mingw" package) risks ABI
      mismatches independent of the underlying issue this script exists to debug.
    - Go: installed via Scoop if missing.

  Requires: Scoop already installed (https://scoop.sh). If you don't have it:
    Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
    Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

.PARAMETER RepoUrl
  Git URL to clone. Defaults to the yetanotherchris/zolam GitHub repo.

.PARAMETER Branch
  Branch to check out after cloning. Defaults to the current PR branch.

.PARAMETER WorkDir
  Directory to clone into. Defaults to .\zolam-windows-build under the current
  directory.

.EXAMPLE
  .\build-windows-release.ps1

.EXAMPLE
  .\build-windows-release.ps1 -Branch main -WorkDir C:\src\zolam
#>
[CmdletBinding()]
param(
    [string]$RepoUrl = "https://github.com/yetanotherchris/zolam.git",
    [string]$Branch = "claude/gha-build-warnings-i71s7o",
    [string]$WorkDir = (Join-Path (Get-Location) "zolam-windows-build")
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

Write-Step "Installing Git, Go, and Rust via Scoop"
scoop bucket list | Select-String -Quiet '^main$' | Out-Null
if (-not $?) { scoop bucket add main }

foreach ($pkg in @("git", "go", "rustup")) {
    if (-not (scoop list $pkg 2>$null | Select-String $pkg)) {
        scoop install $pkg
    } else {
        Write-Host "  $pkg already installed, skipping"
    }
}

Write-Step "Forcing Rust to the GNU host target (Go's cgo here expects MinGW ABI, not MSVC)"
rustup toolchain install stable-x86_64-pc-windows-gnu
rustup default stable-x86_64-pc-windows-gnu

Write-Step "Checking for MSYS2 (needed for a matching GCC + Tesseract/Leptonica)"
$msys2Bash = "C:\msys64\usr\bin\bash.exe"
if (-not (Test-Path $msys2Bash)) {
    if (-not (scoop list msys2 2>$null | Select-String "msys2")) {
        scoop bucket list | Select-String -Quiet '^extras$' | Out-Null
        if (-not $?) { scoop bucket add extras }
        scoop install extras/msys2
    }
    # Scoop's msys2 shim installs elsewhere; find it.
    $msys2Bash = (Get-Command msys2 -ErrorAction SilentlyContinue)?.Source
    if (-not $msys2Bash) {
        throw "Could not locate msys2 after installing via Scoop. Install MSYS2 manually from https://www.msys2.org and re-run."
    }
}
$msys2Root = Split-Path (Split-Path (Split-Path $msys2Bash))
$mingw64 = Join-Path $msys2Root "mingw64"

Write-Step "Installing GCC/Tesseract/Leptonica via pacman (matches CI's package list)"
& $msys2Bash -lc "pacman -Syu --noconfirm"
& $msys2Bash -lc "pacman -S --noconfirm --needed mingw-w64-x86_64-gcc mingw-w64-x86_64-tesseract-ocr mingw-w64-x86_64-leptonica mingw-w64-x86_64-pkg-config"

# Put MSYS2's mingw64 toolchain on PATH for this session so `gcc`/`g++` resolve
# to it consistently for both the cargo build below and the go build.
$env:Path = "$mingw64\bin;$env:Path"

Write-Step "Cloning $RepoUrl (branch: $Branch) into $WorkDir"
if (Test-Path $WorkDir) {
    Write-Host "  $WorkDir already exists, pulling latest instead of re-cloning"
    git -C $WorkDir fetch origin $Branch
    git -C $WorkDir checkout $Branch
    git -C $WorkDir pull origin $Branch
} else {
    git clone --branch $Branch $RepoUrl $WorkDir
}

Push-Location $WorkDir
try {
    Write-Step "Building libtokenizers.a from source (daulet/tokenizers, GNU target)"
    $tokVersionLine = Select-String -Path "src\go.mod" -Pattern "daulet/tokenizers" | Select-Object -First 1
    $tokVersion = ($tokVersionLine.Line -split '\s+')[1]
    Write-Host "  daulet/tokenizers version: $tokVersion"

    $tokSrc = Join-Path $env:TEMP "tokenizers-src"
    if (Test-Path $tokSrc) { Remove-Item -Recurse -Force $tokSrc }
    git clone --depth 1 --branch $tokVersion https://github.com/daulet/tokenizers.git $tokSrc

    cargo build --release --manifest-path "$tokSrc\Cargo.toml" -p tokenizers-ffi --target x86_64-pc-windows-gnu

    $nativeDir = "src\native\tokenizers"
    New-Item -ItemType Directory -Force -Path $nativeDir | Out-Null
    Copy-Item "$tokSrc\target\x86_64-pc-windows-gnu\release\libtokenizers_ffi.a" "$nativeDir\libtokenizers.a" -Force

    Write-Step "Fetching remaining native dependencies (fetchnative sees libtokenizers.a and skips its download)"
    Push-Location src
    try {
        go run ./tools/fetchnative

        Write-Step "Building zolam.exe (CGO_ENABLED=1)"
        # Same -Bdynamic fix as CI: MSYS2's leptonica package ships only a
        # .dll.a import lib, and DuckDB's own #cgo LDFLAGS forces static-only
        # library search later in the link — -Bdynamic keeps leptonica/tesseract
        # resolvable regardless. This gets you to the SAME undefined-reference
        # wall CI hit (go-fitz's MuPDF / DuckDB's C++ static libs) unless you've
        # sorted out a toolchain that resolves those — that's the actual reason
        # this build is manual right now, not something this script fixes.
        $env:CGO_ENABLED = "1"
        $env:CGO_CFLAGS = "-I$mingw64\include"
        $env:CGO_CXXFLAGS = "-I$mingw64\include"
        $env:CGO_LDFLAGS = "-Wl,-Bdynamic -L$mingw64\lib"

        go build -ldflags="-X main.version=dev -s -w" -o zolam.exe ./cmd/zolam/
    } finally {
        Pop-Location
    }

    Write-Step "Copying zolam.exe to artifacts\zolam-windows-amd64.exe"
    New-Item -ItemType Directory -Force -Path "artifacts" | Out-Null
    Copy-Item "src\zolam.exe" "artifacts\zolam-windows-amd64.exe" -Force

    Write-Host ""
    Write-Host "Build succeeded: $WorkDir\artifacts\zolam-windows-amd64.exe" -ForegroundColor Green
    Write-Host "Review it, then commit and push it yourself, e.g.:" -ForegroundColor Yellow
    Write-Host "  cd `"$WorkDir`""
    Write-Host "  git add artifacts/zolam-windows-amd64.exe"
    Write-Host "  git commit -m `"Add manually-built windows-amd64 release binary`""
    Write-Host "  git push origin $Branch"
} finally {
    Pop-Location
}
