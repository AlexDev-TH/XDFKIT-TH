# Builds build\bin\XDFKIT-TH.exe for 64-bit Windows.
#   powershell -File build.ps1
# Needs Go, Node.js and the Wails CLI (go install github.com/wailsapp/wails/v2/cmd/wails@latest).

Set-Location -LiteralPath $PSScriptRoot

# The engine reports the version of the xdfkit release it comes from.
$engine = (git -C .. describe --tags --match 'v[0-9]*' --always) -replace '^v', ''
if (-not $engine) { $engine = 'dev' }

$wails = Join-Path (go env GOPATH) 'bin\wails.exe'
if (-not (Test-Path -LiteralPath $wails)) { $wails = 'wails' }

& $wails build -platform windows/amd64 -ldflags "-X go.nyet.org/xdfkit/api.Version=$engine"
exit $LASTEXITCODE
