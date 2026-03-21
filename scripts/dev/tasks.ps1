param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("fmt", "test", "build", "check")]
    [string]$Task
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

Push-Location $RepoRoot

try {
    switch ($Task) {
        "fmt" {
            go fmt ./...
        }
        "test" {
            go test ./...
        }
        "build" {
            New-Item -ItemType Directory -Force -Path bin | Out-Null
            go build -o ./bin/api.exe ./cmd/api
            go build -o ./bin/worker.exe ./cmd/worker
        }
        "check" {
            & $PSCommandPath -Task fmt
            & $PSCommandPath -Task test
            & $PSCommandPath -Task build
        }
    }
}
finally {
    Pop-Location
}
