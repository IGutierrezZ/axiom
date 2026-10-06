package sdd

import (
	"crypto/sha256"
	"encoding/hex"
)

// historicalOpenCodePluginDigests lists the sha256 of every embedded revision of
// the managed OpenCode plugins that spawned the retired `gentle-ai` binary,
// keyed by plugin file and asset generation ("v1" is opencode/plugins/, "v2" is
// opencode/plugins-v2/). A copy installed under OpenCode v2 whose bytes match one
// of them is an unmodified earlier managed copy, so it is safe to replace with
// the current asset; any other bytes are treated as user-owned and preserved.
// Digests come from `git show <commit>:<asset> | sha256sum` over the full history
// of each asset path; the plugins never lived under a different path.
var historicalOpenCodePluginDigests = map[string]map[string][]string{
	"skill-registry.ts": {
		"v1": {
			"12dcf0eb388ebbb000d8ac318b906a3ab3f7880049fcc0ceed2012143271c4cd", // 8c2f2633
			"5225a5fa9addd6cf8263b645b824c4c4dc1eb4b3b5c740f15ce50de74aab5729", // a1bffc1d
			"af143abb8250087431110739af3e6ae7b16c963f200968b777e1f38b3e58d8ff", // 0efb8b92
			"b24f83e90d455807dbd122f12a953952e7ff6550e3a0d733030c22843f0ce148", // 682d3fe4
			"d402d2414b3c9ec5b31cad6f85787d152037e76148c3cea18fd443c1d7f5eab4", // e23b27b6
		},
		"v2": {
			"c2b4382c014c2c15d4b18c70df8d65c272e557d2639e2b2f1ad8d9236142dc48", // 5a395e45
		},
	},
	"opencode-review-transport.ts": {
		"v1": {
			"05e60cfdce94b017c569fa1752c02938cfd0aebedd4a6d86d2430ed2a4808653", // 7cdcfe57
			"492cab42d786701eb6fa92c4da90dcca14b6408f52ae98af2398ed27131919c8", // 8b98b26a
			"5b70f1f43e792397f06915f889add37bbb758e012b46990680fe62d1163d9dec", // 341cc531
			"82d28ef92c13f2ce2aab5a671896d9c9016b2b9b0776df8e2ff8f5f8b00f944f", // 0c8f8666
			"aa2f0beffddc3496a7e5ddd9eb26d963be7efab31123fd65e7126d478028f222", // 9589e493
			"ea4a9e12442fd03e201a170fca71a2b722473d36e9645e13eef226a0a7c253fd", // a36736a9
			"fa4c1674d8897371341088548e1c4be9098be03f737e97878d008327f6b38e66", // 229332a8
		},
		"v2": {
			"886f5c89a6da4ae1b378aed39cbc7f90b0e4dc5c92350660db015287fb3c2445", // 5a395e45
		},
	},
}

// isHistoricalOpenCodePlugin reports whether data is a known earlier managed
// revision of the named plugin, in either asset generation.
func isHistoricalOpenCodePlugin(name string, data []byte) bool {
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	for _, digests := range historicalOpenCodePluginDigests[name] {
		for _, known := range digests {
			if known == digest {
				return true
			}
		}
	}
	return false
}
