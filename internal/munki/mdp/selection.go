package mdp

import (
	"context"
	"log/slog"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/grant"
)

// redirectTTL is how long a minted grant stays fresh. It is a redirect-freshness
// window, not a download cap: each request is authorized at its start, and a
// resume re-requests the Munki package URL for a fresh grant.
const redirectTTL = 15 * time.Minute

// SelectionRequest is one authorized package download awaiting a location. The
// integrity fields bind the minted grant to specific bytes.
type SelectionRequest struct {
	ClientIP              string
	PackageID             int64
	InstallerItemLocation string
	SHA256                string
	SizeBytes             int64
}

// SelectRedirect returns a redirect URL to a matching distribution point, or
// ok=false to fall back to Woodstar-direct delivery. Every outcome is logged.
func (s *Store) SelectRedirect(ctx context.Context, req SelectionRequest) (string, bool) {
	addr, err := netip.ParseAddr(req.ClientIP)
	if err != nil {
		s.logDecision(ctx, req, 0, "primary_no_client_ip", slog.LevelDebug)
		return "", false
	}

	point, err := s.ResolveForClient(ctx, addr, req.PackageID)
	if err != nil {
		s.logger.WarnContext(ctx, "munki distribution selection failed",
			"operation", "select",
			"package_id", req.PackageID,
			"client_ip", req.ClientIP,
			"err", err,
		)
		return "", false
	}
	if point == nil {
		s.logDecision(ctx, req, 0, "primary_no_match", slog.LevelDebug)
		return "", false
	}

	token, err := grant.Sign([]byte(point.Key), grant.Claims{
		Exp:                   time.Now().Add(redirectTTL).Unix(),
		PackageID:             req.PackageID,
		InstallerItemLocation: req.InstallerItemLocation,
		SHA256:                req.SHA256,
		SizeBytes:             req.SizeBytes,
		DistributionPointID:   point.ID,
	})
	if err != nil {
		s.logger.WarnContext(ctx, "munki distribution grant signing failed",
			"operation", "select",
			"package_id", req.PackageID,
			"distribution_point_id", point.ID,
			"err", err,
		)
		return "", false
	}

	redirect, err := grantURL(point.ClientBaseURL, req.InstallerItemLocation, token)
	if err != nil {
		s.logger.WarnContext(ctx, "munki distribution redirect URL invalid",
			"operation", "select",
			"distribution_point_id", point.ID,
			"err", err,
		)
		return "", false
	}
	s.logDecision(ctx, req, point.ID, "selected_mdp", slog.LevelInfo)
	return redirect, true
}

func (s *Store) logDecision(
	ctx context.Context,
	req SelectionRequest,
	pointID int64,
	reason string,
	level slog.Level,
) {
	s.logger.Log(ctx, level, "munki distribution decision",
		"operation", "select",
		"reason", reason,
		"package_id", req.PackageID,
		"client_ip", req.ClientIP,
		"distribution_point_id", pointID,
	)
}

func grantURL(clientBaseURL string, installerItemLocation string, token string) (string, error) {
	base, err := url.Parse(strings.TrimRight(clientBaseURL, "/") +
		"/munki/pkgs/" + escapePath(installerItemLocation))
	if err != nil {
		return "", err
	}
	values := base.Query()
	values.Set("cap", token)
	base.RawQuery = values.Encode()
	return base.String(), nil
}

func escapePath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
