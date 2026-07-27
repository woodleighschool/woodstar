package events

import (
	"cmp"
	"slices"
)

// ingestPlan separates Santa's client event order from PostgreSQL lock order.
// Shared rows and relationships are deduplicated and sorted so every upload
// acquires overlapping locks in the same order. Occurrence rows remain in
// client order because they do not contend on unique shared keys.
type ingestPlan struct {
	executables            []executableWrite
	signingChains          []signingChainWrite
	certificates           []signingChainEntry
	bundles                []bundleWrite
	executableSigningLinks []executableSigningLink
	bundleExecutableLinks  []bundleExecutableLink
	executionEvents        []ExecutionEventInput
	fileAccessEvents       []FileAccessEventInput
	standaloneEvents       []StandaloneRuleCreationEventInput
	bundleRequestHashes    []string
}

type ingestPlanBuilder struct {
	executables            map[string]executableWrite
	signingChains          map[string]signingChainWrite
	certificates           map[string]signingChainEntry
	bundles                map[string]bundleWrite
	executableSigningLinks map[executableSigningLink]struct{}
	bundleExecutableLinks  map[bundleExecutableLink]struct{}
	bundleRequestHashes    map[string]struct{}
}

type executableSigningLink struct {
	executableSHA256   string
	signingChainSHA256 string
}

type bundleExecutableLink struct {
	bundleSHA256     string
	executableSHA256 string
}

func newIngestPlan(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneEvents []StandaloneRuleCreationEventInput,
) ingestPlan {
	builder := ingestPlanBuilder{
		executables:            make(map[string]executableWrite),
		signingChains:          make(map[string]signingChainWrite),
		certificates:           make(map[string]signingChainEntry),
		bundles:                make(map[string]bundleWrite),
		executableSigningLinks: make(map[executableSigningLink]struct{}),
		bundleExecutableLinks:  make(map[bundleExecutableLink]struct{}),
		bundleRequestHashes:    make(map[string]struct{}),
	}
	for _, event := range executionEvents {
		builder.addExecutionEvent(event)
	}
	return builder.build(executionEvents, fileAccessEvents, standaloneEvents)
}

func (b *ingestPlanBuilder) addExecutionEvent(event ExecutionEventInput) {
	// Executable and certificate metadata retain the last occurrence in the
	// upload, matching the former event-at-a-time upserts.
	b.executables[event.FileSHA256] = executableWriteFromEvent(event)
	b.addSigningChain(event.FileSHA256, event.SigningChain)

	if event.BundleHash == "" {
		return
	}
	bundle := b.bundles[event.BundleHash]
	bundle.merge(event)
	b.bundles[event.BundleHash] = bundle
	b.bundleExecutableLinks[bundleExecutableLink{
		bundleSHA256:     event.BundleHash,
		executableSHA256: event.FileSHA256,
	}] = struct{}{}
	if event.Decision != ExecutionDecisionBundleBinary {
		b.bundleRequestHashes[event.BundleHash] = struct{}{}
	}
}

func (b *ingestPlanBuilder) addSigningChain(executableSHA256 string, chain []CertificateInput) {
	entries := signingChainEntries(chain)
	if len(entries) == 0 {
		return
	}
	chainSHA256 := signingChainHash(entries)
	b.signingChains[chainSHA256] = signingChainWrite{
		SHA256:  chainSHA256,
		Entries: entries,
	}
	b.executableSigningLinks[executableSigningLink{
		executableSHA256:   executableSHA256,
		signingChainSHA256: chainSHA256,
	}] = struct{}{}
	for _, entry := range entries {
		b.certificates[entry.SHA256] = entry
	}
}

func (b *ingestPlanBuilder) build(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneEvents []StandaloneRuleCreationEventInput,
) ingestPlan {
	return ingestPlan{
		executables:            sortedMapValues(b.executables),
		signingChains:          sortedMapValues(b.signingChains),
		certificates:           sortedMapValues(b.certificates),
		bundles:                sortedMapValues(b.bundles),
		executableSigningLinks: sortedExecutableSigningLinks(b.executableSigningLinks),
		bundleExecutableLinks:  sortedBundleExecutableLinks(b.bundleExecutableLinks),
		executionEvents:        executionEventRows(executionEvents),
		fileAccessEvents:       slices.Clone(fileAccessEvents),
		standaloneEvents:       slices.Clone(standaloneEvents),
		bundleRequestHashes:    sortedMapKeys(b.bundleRequestHashes),
	}
}

func executionEventRows(events []ExecutionEventInput) []ExecutionEventInput {
	rows := make([]ExecutionEventInput, 0, len(events))
	for _, event := range events {
		if event.Decision != ExecutionDecisionBundleBinary {
			rows = append(rows, event)
		}
	}
	return rows
}

func sortedMapValues[T any](values map[string]T) []T {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	sorted := make([]T, 0, len(keys))
	for _, key := range keys {
		sorted = append(sorted, values[key])
	}
	return sorted
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func sortedExecutableSigningLinks(values map[executableSigningLink]struct{}) []executableSigningLink {
	links := make([]executableSigningLink, 0, len(values))
	for link := range values {
		links = append(links, link)
	}
	slices.SortFunc(links, func(left executableSigningLink, right executableSigningLink) int {
		if n := cmp.Compare(left.executableSHA256, right.executableSHA256); n != 0 {
			return n
		}
		return cmp.Compare(left.signingChainSHA256, right.signingChainSHA256)
	})
	return links
}

func sortedBundleExecutableLinks(values map[bundleExecutableLink]struct{}) []bundleExecutableLink {
	links := make([]bundleExecutableLink, 0, len(values))
	for link := range values {
		links = append(links, link)
	}
	slices.SortFunc(links, func(left bundleExecutableLink, right bundleExecutableLink) int {
		if n := cmp.Compare(left.bundleSHA256, right.bundleSHA256); n != 0 {
			return n
		}
		return cmp.Compare(left.executableSHA256, right.executableSHA256)
	})
	return links
}
