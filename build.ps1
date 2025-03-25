$targets = @(
    @{os="windows"; arch="amd64"; ext=".exe"}
    @{os="windows"; arch="386"; ext=".exe"}
    @{os="windows"; arch="arm"; ext=".exe"}
    @{os="windows"; arch="arm64"; ext=".exe"}
)

foreach ($target in $targets) {
    $env:GOOS = $target.os
    $env:GOARCH = $target.arch
    $output = "bin/BPB-Wizard-$($target.arch)$($target.ext)"
    go build -o $output
    Write-Host "Built: $output"
}
