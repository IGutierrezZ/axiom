package cli

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
)

// managedAssetDigest returns a content digest of the managed assets this
// binary embeds. It deliberately does NOT use the review capabilities build
// identity: that identity carries vcs.revision, module version, and ldflags,
// so two binaries built from the same source disagree even when they embed
// byte-identical assets. Comparing build identities would therefore declare
// perfectly current assets outdated after any rebuild, and a test binary
// (which Go does not stamp with VCS settings) could never agree with a
// released one. The digest compares the thing #2685 is actually about.
func managedAssetDigest() (string, error) {
	digest, err := assets.ManagedDigest()
	if err != nil {
		return "", fmt.Errorf("derive managed asset digest: %w", err)
	}
	return digest, nil
}
