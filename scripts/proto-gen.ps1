# proto-gen.ps1 - Generate Go code from proto files
# Prerequisites: protoc, go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#                go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
$ErrorActionPreference = "Stop"

# Add Go bin to PATH (protoc-gen-go, protoc-gen-go-grpc)
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
$genDir = Join-Path $projectPath "pkg\gen"

Write-Host "Project: $projectPath" -ForegroundColor Cyan
Write-Host "Proto: $protoDir" -ForegroundColor Cyan
Write-Host "Output: $genDir" -ForegroundColor Cyan

if (-Not (Test-Path $protoDir)) {
    Write-Host "ERROR: Proto dir not found" -ForegroundColor Red
    exit 1
}

New-Item -ItemType Directory -Path $genDir -Force | Out-Null

$protoFiles = @("common.proto", "video.proto", "client_info.proto")
foreach ($f in $protoFiles) {
    $path = Join-Path $protoDir $f
    if (Test-Path $path) {
        Write-Host "Processing $f..." -ForegroundColor Green
        protoc -I $protoDir --go_out=$genDir --go_opt=paths=source_relative `
            --go-grpc_out=$genDir --go-grpc_opt=paths=source_relative $path
        if ($LASTEXITCODE -ne 0) { exit 1 }
    }
}

Write-Host "Done. Check pkg/gen/" -ForegroundColor Green
