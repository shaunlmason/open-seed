# open-seed bootstrap shim, Windows twin (decision 7.5). Downloads the pinned
# engine release, verifies its SHA-256 against .seed/engine.lock, caches it
# outside the repo, and runs it. See scripts/seed for the POSIX version.
$ErrorActionPreference = 'Stop'

function Fail($msg) { Write-Error "seed bootstrap: $msg"; exit 1 }

# Locate the repo root (the directory containing .seed\).
$root = (Get-Location).Path
while (-not (Test-Path (Join-Path $root '.seed'))) {
  $parent = Split-Path $root -Parent
  if (-not $parent -or $parent -eq $root) { Fail "no .seed directory found above $(Get-Location)" }
  $root = $parent
}
$lockPath = Join-Path $root '.seed\engine.lock'
if (-not (Test-Path $lockPath)) { Fail "missing $lockPath (the engine pin)" }

$lock = @{}
foreach ($line in Get-Content $lockPath) {
  if ($line -match '^\s*#' -or $line -notmatch '\S') { continue }
  $parts = $line -split '\s+', 2
  if ($parts.Count -eq 2) { $lock[$parts[0]] = $parts[1].Trim() }
}

if ($env:SEED_ENGINE) {
  if (-not (Test-Path $env:SEED_ENGINE)) { Fail "SEED_ENGINE=$($env:SEED_ENGINE) not found" }
  & $env:SEED_ENGINE @args; exit $LASTEXITCODE
}
if ($lock['vendor']) {
  if (-not (Test-Path $lock['vendor'])) { Fail "vendored engine $($lock['vendor']) (from engine.lock) not found" }
  & $lock['vendor'] @args; exit $LASTEXITCODE
}

$version = $lock['version']; $repo = $lock['repo']
if (-not $version -or -not $repo) { Fail 'engine.lock missing version/repo' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  default { Fail "unsupported architecture $($env:PROCESSOR_ARCHITECTURE) — set SEED_ENGINE" }
}
$want = $lock["sha256_windows_$arch"]
if (-not $want) { Fail "engine.lock has no sha256 for windows_$arch" }

$cache = Join-Path $env:LOCALAPPDATA "open-seed\engine\$version\windows_$arch"
$bin = Join-Path $cache 'seed.exe'
if (-not (Test-Path $bin)) {
  New-Item -ItemType Directory -Force -Path $cache | Out-Null
  $verNum = $version.TrimStart('v')
  $archive = "seed_${verNum}_windows_$arch.zip"
  $url = "https://github.com/$repo/releases/download/$version/$archive"
  $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
  New-Item -ItemType Directory -Path $tmp | Out-Null
  try {
    try { Invoke-WebRequest -Uri $url -OutFile (Join-Path $tmp $archive) -UseBasicParsing }
    catch { Fail "download failed: $url — check network access, or vendor the binary (engine.lock 'vendor <path>' / SEED_ENGINE)" }
    $got = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp $archive)).Hash.ToLower()
    if ($got -ne $want.ToLower()) { Fail "SHA-256 mismatch for ${archive}: got $got, lock says $want — refusing to run an unverified engine" }
    Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath $tmp
    Move-Item (Join-Path $tmp 'seed.exe') $bin
  } finally { Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue }
}
& $bin @args
exit $LASTEXITCODE
