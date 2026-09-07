# CLAUDE.md — wakelab

## Design principle (non-negotiable)

Keep it **simple and fast**, and **easy to use by a human and an LLM alike**.
Do not let the tool grow complex. Every change is judged against this.

## What this is

`wakelab` is a Go CLI (binary: `wake`) to wake, sleep, and control a group of
home servers over Wake-on-LAN and SSH. Config lives in ssh-config-style blocks
in `~/.wol_hosts`, with groups in `~/.wol_groups`.

## Commands (run these, not raw go)

- `make build`   — build `bin/wake`
- `make test`    — run all tests (race detector on)
- `make cover`   — tests + coverage summary
- `make lint`    — golangci-lint
- `make fmt`     — gofmt the tree
- `make install` — build and copy to ~/.local/bin

## Conventions

- **Conventional Commits**, atomic. Never mix formatting and logic in one commit.
- PRs kept to ~600 lines (source and tests counted separately).
- Production code lives under `internal/`; the entrypoint under `cmd/wake/`.
- **Inject side effects behind interfaces.** Running `ssh`, `crontab`, sending a
  UDP packet, or reading the clock goes through a small interface so commands are
  tested with a fake — no real network in unit tests.
- Every feature ships with a test. TDD: write the failing test first.

## Do not

- Add runtime dependencies (the binary must stay self-contained; WOL is native,
  SSH shells out to the system `ssh`).
- Add interactive-only behavior as a default — it breaks non-TTY and LLM use.
  Make such features explicit and opt-in with a non-interactive fallback.
