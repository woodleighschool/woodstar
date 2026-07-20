// Package syncstate tracks versioned Santa rule synchronization per host.
package syncstate

type SyncType string

const (
	SyncTypeNormal   SyncType = "normal"
	SyncTypeClean    SyncType = "clean"
	SyncTypeCleanAll SyncType = "clean_all"
)

type RuleCounts struct {
	Binary      uint32
	Certificate uint32
	TeamID      uint32
	SigningID   uint32
	CDHash      uint32
	Compiler    uint32
	Transitive  uint32
}

func (counts RuleCounts) MatchesReported(reported RuleCounts) bool {
	return counts.Binary == reported.binaryWithoutTransitive() &&
		counts.Certificate == reported.Certificate &&
		counts.TeamID == reported.TeamID &&
		counts.SigningID == reported.SigningID &&
		counts.CDHash == reported.CDHash &&
		counts.Compiler == reported.Compiler
}

func (counts RuleCounts) binaryWithoutTransitive() uint32 {
	return counts.Binary - counts.Transitive
}
