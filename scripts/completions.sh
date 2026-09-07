#!/bin/sh
# Generate shell completions for the release archives (goreleaser pre-hook).
set -eu
rm -rf completions
mkdir -p completions
go run ./cmd/wake completion bash > completions/wake.bash
go run ./cmd/wake completion zsh > completions/_wake
go run ./cmd/wake completion fish > completions/wake.fish
