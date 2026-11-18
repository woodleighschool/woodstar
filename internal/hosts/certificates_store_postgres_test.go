//go:build postgres

package hosts

import "testing"

func TestCertificateSnapshotReconcilesByHostAndCertificate(t *testing.T) {
	store, ctx := newPostgresHostStore(t)
	host := enrollTestHost(t, ctx, store, "cert-snapshot", nil)
	other := enrollTestHost(t, ctx, store, "cert-other-host", nil)
	cert := HostCertificate{SHA1: "hash-a", Source: "system", CommonName: "A", Path: "/a", Subject: CertificateName{Organization: "Example"}}
	for _, id := range []int64{host.ID, other.ID} {
		if err := store.ReplaceCertificates(ctx, id, []HostCertificate{cert}); err != nil {
			t.Fatal(err)
		}
	}

	load := func(hostID int64) HostCertificate {
		t.Helper()
		certs, err := store.ListCertificates(ctx, hostID)
		if err != nil || len(certs) != 1 {
			t.Fatalf("load certificates = %+v, %v", certs, err)
		}
		return certs[0]
	}
	version := func() string {
		t.Helper()
		var xmin string
		if err := store.pool.QueryRow(ctx, "SELECT xmin::text FROM host_certificates WHERE host_id = $1", host.ID).Scan(&xmin); err != nil {
			t.Fatal(err)
		}
		return xmin
	}
	first := load(host.ID)
	before := version()
	if err := store.ReplaceCertificates(ctx, host.ID, []HostCertificate{cert}); err != nil {
		t.Fatal(err)
	}
	if version() != before {
		t.Fatal("replay rewrote an unchanged certificate")
	}

	changed := cert
	changed.Path = "/b"
	if err := store.ReplaceCertificates(ctx, host.ID, []HostCertificate{cert, changed}); err != nil {
		t.Fatal(err)
	}
	if got := load(host.ID); got.ID != first.ID || got.Path != changed.Path || got.Subject != cert.Subject {
		t.Fatalf("updated certificate = %+v", got)
	}
	if err := store.ReplaceCertificates(ctx, host.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := store.ListCertificates(ctx, host.ID); err != nil || len(got) != 0 {
		t.Fatalf("empty snapshot = %+v, %v", got, err)
	}
	if got := load(other.ID); got.Path != cert.Path {
		t.Fatalf("other host's certificate = %+v", got)
	}
}
