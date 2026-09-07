#!/bin/sh
# Integration checks for the built wake binary. POSIX sh (runs on busybox ash).
# Usage: sh tests/integration.sh /path/to/wake
# Everything external (ssh, crontab, ping, ip) is shimmed on PATH inside a
# temp HOME, so the script exercises the real binary and its shell-outs with
# no real network or SSH.
set -eu

WAKE=$(readlink -f "${1:?usage: integration.sh /path/to/wake}")

HOME=$(mktemp -d)
export HOME
mkdir -p "$HOME/bin" "$HOME/.ssh"
PATH="$HOME/bin:$PATH"
export PATH

fail() { echo "FAIL: $*" >&2; exit 1; }

# --- shims -------------------------------------------------------------
cat > "$HOME/bin/ssh" <<'EOF'
#!/bin/sh
if [ "${1:-}" = "-G" ]; then
    echo "user diego"
    echo "hostname 127.0.0.1"
    exit 0
fi
exit 0
EOF

cat > "$HOME/bin/crontab" <<'EOF'
#!/bin/sh
CRON="$HOME/.fakecron"
case "${1:-}" in
    -l) [ -f "$CRON" ] || exit 1; cat "$CRON" ;;
    -)  cat > "$CRON" ;;
    *)  exit 1 ;;
esac
EOF

cat > "$HOME/bin/ping" <<'EOF'
#!/bin/sh
exit 1
EOF

cat > "$HOME/bin/ip" <<'EOF'
#!/bin/sh
case "$*" in
    *"route show to default"*) echo "default via 10.0.0.1 dev eth0" ;;
    *"addr show dev eth0"*)    echo "2: eth0 inet 10.0.0.5/24 brd 10.0.0.255 scope global" ;;
    *neigh*)                   : ;;
esac
exit 0
EOF
chmod +x "$HOME/bin/ssh" "$HOME/bin/crontab" "$HOME/bin/ping" "$HOME/bin/ip"

# --- config ------------------------------------------------------------
cat > "$HOME/.wol_hosts" <<'EOF'
Host testbox
    Mac        aa:bb:cc:dd:ee:ff
    Broadcast  127.0.0.1
EOF
echo "grp testbox" > "$HOME/.wol_groups"
printf 'Host testbox box2\n' > "$HOME/.ssh/config"

# --- checks ------------------------------------------------------------
"$WAKE" --version | grep -q "wake version" || fail "--version"
"$WAKE" --help > /dev/null || fail "--help"

"$WAKE" ls | grep -q "testbox" || fail "ls"
"$WAKE" status --json | grep -q '"host":"testbox"' || fail "status --json"
"$WAKE" status grp > /dev/null || fail "status <group>"

"$WAKE" add box2 --mac 11-22-33-44-55-66 --port 8080 | grep -q "Added 'box2'" || fail "add"
grep -q "11:22:33:44:55:66" "$HOME/.wol_hosts" || fail "add persisted"
"$WAKE" edit box2 --broadcast 127.0.0.1 > /dev/null || fail "edit"
grep -q "127.0.0.1" "$HOME/.wol_hosts" || fail "edit persisted"

"$WAKE" group add g2 --devices box2 > /dev/null || fail "group add"
"$WAKE" groups | grep -q "g2" || fail "groups"
"$WAKE" group edit g2 --add testbox > /dev/null || fail "group edit"
"$WAKE" group rm g2 > /dev/null || fail "group rm"
"$WAKE" rm box2 > /dev/null || fail "rm"
if grep -q "box2" "$HOME/.wol_hosts"; then fail "rm persisted"; fi

"$WAKE" schedule add testbox 07:00 | grep -q "Scheduled" || fail "schedule add"
"$WAKE" schedule list | grep -q "testbox" || fail "schedule list"
"$WAKE" schedule rm 1 | grep -q "Removed schedule 1" || fail "schedule rm"

# Real UDP send, fully local (Broadcast is 127.0.0.1).
"$WAKE" testbox | grep -q "Waking 'testbox'" || fail "wake"
"$WAKE" aa:bb:cc:dd:ee:ff --broadcast 127.0.0.1 | grep -q "Waking aa:bb:cc:dd:ee:ff" || fail "wake raw mac"

"$WAKE" scan --subnet 10.0.0.0/24 --json | grep -q '^\[' || fail "scan --json"
"$WAKE" doctor > /dev/null || fail "doctor"

"$WAKE" completion bash > /dev/null || fail "completion bash"

echo "OK: all integration checks passed on $(uname -a)"
