package cli

import "testing"

// The help should teach the remote-wake recipe: an always-on relay reached over
// Tailscale/VPN/SSH.
func TestRootHelpExplainsRemoteWake(t *testing.T) {
	env := newTestEnv(t, "", "")
	root := NewRootCmd(env.app)
	mustContain(t, root.Long, "Remote wake", "always-on", "Tailscale")
}
