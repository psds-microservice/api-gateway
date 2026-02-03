# proto-gen.ps1 - Generate Go code from proto (like user-service, no helpy)
# Prerequisites: protoc, protoc-gen-go, protoc-gen-go-grpc
$ErrorActionPreference = "Stop"

$goBin = "$env:USERPROFILE\go\bin"
if ($env:GOPATH) { $goBin = "$env:GOPATH\bin" }
if (Test-Path $goBin) { $env:Path = "$goBin;$env:Path" }

try {
    $projectPath = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
} catch {
    $projectPath = (Get-Location).Path
}
Set-Location $projectPath

$protoDir = Join-Path $projectPath "pkg\api_gateway"
$thirdParty = Join-Path $projectPath "third_party"
$genDir = Join-Path $projectPath "pkg\gen"
$module = "github.com/psds-microservice/api-gateway"

Write-Host "Project: $projectPath" -ForegroundColor Cyan
Write-Host "Proto: $protoDir, third_party: $thirdParty, Output: $genDir" -ForegroundColor Cyan

if (-Not (Test-Path $protoDir)) {
    Write-Host "ERROR: Proto dir not found" -ForegroundColor Red
    exit 1
}

New-Item -ItemType Directory -Path $genDir -Force | Out-Null

$protoFiles = Get-ChildItem -Path $protoDir -Filter "*.proto"
foreach ($f in $protoFiles) {
    Write-Host "Processing $($f.Name)..." -ForegroundColor Green
    protoc -I $protoDir -I $thirdParty --go_out=. --go_opt=module=$module --go-grpc_out=. --go-grpc_opt=module=$module $f.FullName
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Write-Host "Done. Check pkg/gen/" -ForegroundColor Green
