// Package grant mints and verifies Munki distribution point download grants.
//
// A grant is a storage capability signed with a distribution point's per-DP
// key. Woodstar mints one when it redirects a Munki client to a distribution
// point; the worker verifies it offline with the same key. The package is a
// DB-free leaf so both the server and the worker can share one claims
// definition without the worker depending on the database.
package grant

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

// Claims authorizes one package download from one distribution point. SHA256
// and SizeBytes bind the grant to specific bytes so the worker can re-check its
// mirror before serving.
type Claims struct {
	Op                    string `json:"op"`
	Exp                   int64  `json:"exp"`
	PackageID             int64  `json:"package_id"`
	InstallerItemLocation string `json:"installer_item_location"`
	SHA256                string `json:"sha256"`
	SizeBytes             int64  `json:"size_bytes"`
	DistributionPointID   int64  `json:"distribution_point_id"`
}

// Sign returns a grant token signed with the distribution point's key. The
// caller sets Exp; the operation is always a read.
func Sign(key []byte, claims Claims) (string, error) {
	claims.Op = capability.OpGet
	return capability.Sign(key, claims)
}

// Verify checks a grant token against the distribution point's key and returns
// its claims. The caller still matches the package id and re-checks integrity.
func Verify(key []byte, token string, now time.Time) (Claims, error) {
	return capability.Verify[Claims](key, token, capability.OpGet, now)
}
