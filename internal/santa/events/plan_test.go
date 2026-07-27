package events

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestIngestPlanOrdersSharedWritesIndependentOfEventOrder(t *testing.T) {
	t.Parallel()

	events := []ExecutionEventInput{
		{
			FileSHA256: "executable-b",
			FileName:   "B",
			OccurredAt: time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC),
			Decision:   ExecutionDecisionAllowBinary,
			BundleHash: "bundle-a",
			BundleName: "Bundle A",
			SigningChain: []CertificateInput{
				{SHA256: "certificate-b", CommonName: "B"},
				{SHA256: "certificate-a", CommonName: "A"},
			},
		},
		{
			FileSHA256: "executable-a",
			FileName:   "A",
			OccurredAt: time.Date(2026, 7, 27, 10, 1, 0, 0, time.UTC),
			Decision:   ExecutionDecisionAllowBinary,
			BundleHash: "bundle-b",
			BundleName: "Bundle B",
			SigningChain: []CertificateInput{
				{SHA256: "certificate-a", CommonName: "A"},
			},
		},
	}

	forward := newIngestPlan(events, nil, nil)
	reversedEvents := slices.Clone(events)
	slices.Reverse(reversedEvents)
	reversed := newIngestPlan(reversedEvents, nil, nil)

	forwardOrder := sharedWriteOrder(forward)
	reversedOrder := sharedWriteOrder(reversed)
	if !reflect.DeepEqual(forwardOrder, reversedOrder) {
		t.Fatalf("shared write order differs by upload order:\nforward: %#v\nreverse: %#v", forwardOrder, reversedOrder)
	}
	if got := executableWriteHashes(forward.executables); !slices.Equal(got, []string{"executable-a", "executable-b"}) {
		t.Fatalf("executable order = %v, want SHA-256 order", got)
	}
	if got := certificateWriteHashes(forward.certificates); !slices.Equal(got, []string{"certificate-a", "certificate-b"}) {
		t.Fatalf("certificate order = %v, want SHA-256 order", got)
	}
	if got := bundleWriteHashes(forward.bundles); !slices.Equal(got, []string{"bundle-a", "bundle-b"}) {
		t.Fatalf("bundle order = %v, want SHA-256 order", got)
	}
	if !slices.IsSortedFunc(forward.signingChains, compareSigningChainWrites) {
		t.Fatalf("signing chains are not in SHA-256 order: %+v", forward.signingChains)
	}
}

func TestIngestPlanPreservesRepeatedReferenceSemantics(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC)
	plan := newIngestPlan([]ExecutionEventInput{
		{
			FileSHA256:        "executable",
			FileName:          "Old Name",
			OccurredAt:        occurredAt,
			Decision:          ExecutionDecisionAllowBinary,
			BundleHash:        "bundle",
			BundleName:        "Retained Name",
			BundleBinaryCount: 1,
			SigningChain: []CertificateInput{{
				SHA256:     "certificate",
				CommonName: "Old Certificate",
			}},
		},
		{
			FileSHA256:        "executable",
			FileName:          "New Name",
			OccurredAt:        occurredAt.Add(time.Minute),
			Decision:          ExecutionDecisionBlockBinary,
			BundleHash:        "bundle",
			BundleVersion:     "2.0",
			BundleBinaryCount: 2,
			SigningChain: []CertificateInput{{
				SHA256:     "certificate",
				CommonName: "New Certificate",
			}},
		},
	}, nil, nil)

	if len(plan.executables) != 1 || plan.executables[0].FileName != "New Name" {
		t.Fatalf("executables = %+v, want last occurrence metadata", plan.executables)
	}
	if len(plan.certificates) != 1 || plan.certificates[0].CommonName != "New Certificate" {
		t.Fatalf("certificates = %+v, want last occurrence metadata", plan.certificates)
	}
	if len(plan.bundles) != 1 ||
		plan.bundles[0].Name != "Retained Name" ||
		plan.bundles[0].Version != "2.0" ||
		plan.bundles[0].BinaryCount != 2 {
		t.Fatalf("bundles = %+v, want fieldwise non-empty metadata", plan.bundles)
	}
	if len(plan.executableSigningLinks) != 1 || len(plan.bundleExecutableLinks) != 1 {
		t.Fatalf(
			"relationship counts = %d/%d, want deduplicated links",
			len(plan.executableSigningLinks),
			len(plan.bundleExecutableLinks),
		)
	}
	if len(plan.executionEvents) != 2 {
		t.Fatalf("execution event count = %d, want every occurrence", len(plan.executionEvents))
	}
}

func TestIngestPlanOrdersRelationshipCompositeKeys(t *testing.T) {
	t.Parallel()

	chainA := []CertificateInput{{SHA256: "certificate-a"}}
	chainB := []CertificateInput{{SHA256: "certificate-b"}}
	plan := newIngestPlan([]ExecutionEventInput{
		{
			FileSHA256:   "executable-b",
			OccurredAt:   time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
			Decision:     ExecutionDecisionAllowBinary,
			BundleHash:   "bundle-b",
			SigningChain: chainB,
		},
		{
			FileSHA256:   "executable-a",
			OccurredAt:   time.Date(2026, 7, 27, 12, 1, 0, 0, time.UTC),
			Decision:     ExecutionDecisionAllowBinary,
			BundleHash:   "bundle-b",
			SigningChain: chainB,
		},
		{
			FileSHA256:   "executable-a",
			OccurredAt:   time.Date(2026, 7, 27, 12, 2, 0, 0, time.UTC),
			Decision:     ExecutionDecisionAllowBinary,
			BundleHash:   "bundle-a",
			SigningChain: chainA,
		},
	}, nil, nil)

	chainHashes := []string{
		signingChainHash(signingChainEntries(chainA)),
		signingChainHash(signingChainEntries(chainB)),
	}
	slices.Sort(chainHashes)
	wantSigningLinks := []executableSigningLink{
		{executableSHA256: "executable-a", signingChainSHA256: chainHashes[0]},
		{executableSHA256: "executable-a", signingChainSHA256: chainHashes[1]},
		{executableSHA256: "executable-b", signingChainSHA256: signingChainHash(signingChainEntries(chainB))},
	}
	if !slices.Equal(plan.executableSigningLinks, wantSigningLinks) {
		t.Fatalf("executable signing links = %+v, want %+v", plan.executableSigningLinks, wantSigningLinks)
	}

	wantBundleLinks := []bundleExecutableLink{
		{bundleSHA256: "bundle-a", executableSHA256: "executable-a"},
		{bundleSHA256: "bundle-b", executableSHA256: "executable-a"},
		{bundleSHA256: "bundle-b", executableSHA256: "executable-b"},
	}
	if !slices.Equal(plan.bundleExecutableLinks, wantBundleLinks) {
		t.Fatalf("bundle executable links = %+v, want %+v", plan.bundleExecutableLinks, wantBundleLinks)
	}
}

type sharedWritePlan struct {
	executables            []executableWrite
	signingChains          []signingChainWrite
	certificates           []signingChainEntry
	bundles                []bundleWrite
	executableSigningLinks []executableSigningLink
	bundleExecutableLinks  []bundleExecutableLink
	bundleRequestHashes    []string
}

func sharedWriteOrder(plan ingestPlan) sharedWritePlan {
	return sharedWritePlan{
		executables:            plan.executables,
		signingChains:          plan.signingChains,
		certificates:           plan.certificates,
		bundles:                plan.bundles,
		executableSigningLinks: plan.executableSigningLinks,
		bundleExecutableLinks:  plan.bundleExecutableLinks,
		bundleRequestHashes:    plan.bundleRequestHashes,
	}
}

func executableWriteHashes(writes []executableWrite) []string {
	hashes := make([]string, 0, len(writes))
	for _, write := range writes {
		hashes = append(hashes, write.SHA256)
	}
	return hashes
}

func certificateWriteHashes(writes []signingChainEntry) []string {
	hashes := make([]string, 0, len(writes))
	for _, write := range writes {
		hashes = append(hashes, write.SHA256)
	}
	return hashes
}

func bundleWriteHashes(writes []bundleWrite) []string {
	hashes := make([]string, 0, len(writes))
	for _, write := range writes {
		hashes = append(hashes, write.SHA256)
	}
	return hashes
}

func compareSigningChainWrites(left signingChainWrite, right signingChainWrite) int {
	switch {
	case left.SHA256 < right.SHA256:
		return -1
	case left.SHA256 > right.SHA256:
		return 1
	default:
		return 0
	}
}
