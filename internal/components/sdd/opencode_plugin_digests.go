package sdd

import (
	"crypto/sha256"
	"encoding/hex"
)

// historicalOpenCodePluginDigests lists the sha256 of every earlier embedded
// revision of the managed OpenCode plugins, keyed by plugin file and asset
// generation ("v1" is opencode/plugins/, "v2" is opencode/plugins-v2/). The
// current embedded revision is deliberately not listed. A copy installed under
// OpenCode v2 whose bytes match one of them is an unmodified earlier managed
// copy, so it is safe to replace with the current asset; any other bytes are
// treated as user-owned and preserved.
// Digests come from `git show <commit>:<asset> | sha256sum` over the full history
// of each asset path; the plugins never lived under a different path. A plugin
// that gains a new revision must add the digest of the one it replaces here.
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
	"model-variants.ts": {
		"v1": {
			"1e7e9c8f3b45ba89a9a9de96425e1dd1edcd5b0efd8c9cd7cb4a53c64812c606", // d45d4583
			"47bad2b6a0ebc13cb921174b8b2d61628767b312edd2373a426b6979e5df170d", // 07a455fb
			"603585f4dfb83cf991a16b1b48c58a3bf687e08dbfc9161a9102432c1a7a7c80", // af73567c
			"a54d36130a695ef1383f0e8bb6c559e99a48aceda52170f51a051324321fa741", // 3a30b25f
			"f34c4e9c33848d5c34792a65a36a2279063ccb8e071f24cb6d9b2e83bce6e01b", // a517efde
			"fdcbf9c6e4c398216c32394abf4c1ee2f9ed17eb96b1238c226bf5dd6a08416f", // 27fc25ec
		},
	},
	"sdd-task-result-artifacts.ts": {
		"v1": {
			"0c514e085be33de5871da15bd2b021aea286676c2a4b723c31ba02b8acd8833d", // 7271ff29
			"9440325b6ea325804f4972a993cbb457b7babe0085010a47e16c00763afb9c5f", // cfc87f6e
			"a571b08f05c6f0d3726fbbd0144b387d2b139c4a701f398d7c7d2044878dd3d5", // 3c3b8a02
			"b8f1cdcfe38df960d37d8700eb6a9875335302761bfca1d0d10df02b4c18b7d4", // 3b3ca3d3
			"bfd291ff60c7d186d3e727327a9cc038a8ac4275128bcb8b406cb322858eb6f6", // 229332a8
			"e497b37070f5178fcab110bd964b7fd0af6e299c467eb630093f9c4d9ca7e404", // 7a45a34c
		},
		"v2": {
			"2c49d774b21edaf7bd6ffb9d3447d9dd54828dce6dd846fcc49274173bebd3ca", // 5a395e45
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
