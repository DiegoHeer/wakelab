# wakelab

**Wake and sleep your whole homelab from the CLI.**

`wakelab` provides the `wake` command: Wake-on-LAN plus SSH control for a group
of home servers. Native magic packets (no `wakeonlan` dependency), ssh-config-style
host files, groups, relay wakes for machines on another subnet, readiness waits,
scheduled wakes via cron, LAN scanning, and a config doctor — in one static binary
for Linux and macOS.

## Install

Homebrew (Linux or macOS):

```
brew install DiegoHeer/tap/wakelab
```

Go:

```
go install github.com/DiegoHeer/wakelab/cmd/wake@latest
```

Or grab a binary from the [releases page](https://github.com/DiegoHeer/wakelab/releases).

## Usage

Hosts live in `~/.wol_hosts` as ssh-config-style blocks; groups in `~/.wol_groups`:

```
Host desktop
    Mac        b0:41:6f:16:32:ee
    Broadcast  255.255.255.255
    Via        proxmox          # send the wake from this SSH host (optional)
```

```
wake desktop                  # wake one host
wake minirack --wait          # wake a group, poll until every host is up
wake all                      # wake everything
wake status                   # online/offline table
wake status desktop           # one line; exit 1 when offline
wake status desktop --json    # machine-readable status
wake ls                       # quick host list
wake add nas --ip 192.168.1.50
wake edit nas --via proxmox
wake import-ssh               # import online hosts from ~/.ssh/config
wake group add minirack --devices cluster1 cluster2 cluster3
wake poweroff desktop         # over SSH (asks first; -y skips)
wake suspend desktop          # sleep it — wake it again with 'wake desktop'
wake schedule add server 07:00
wake scan                     # discover devices on your LAN
wake doctor                   # check the config for problems
```

Every command has `--help`. Homebrew installs shell completions automatically;
other installs can generate them with `wake completion bash|zsh|fish`.

## Build from source

```
git clone https://github.com/DiegoHeer/wakelab
cd wakelab
make build
./bin/wake --help
```

## License

GPLv3 — see [LICENSE](LICENSE).
