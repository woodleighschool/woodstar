package syncstate

type SyncType string

const (
	SyncTypeNormal   SyncType = "normal"
	SyncTypeClean    SyncType = "clean"
	SyncTypeCleanAll SyncType = "clean_all"
)

type RuleCounts struct {
	Binary      int32
	Certificate int32
	TeamID      int32
	SigningID   int32
	CDHash      int32
	Compiler    int32
	Transitive  int32
}

func (counts RuleCounts) MatchesReported(reported RuleCounts) bool {
	return counts.Binary == reported.binaryWithoutTransitive() &&
		counts.Certificate == reported.Certificate &&
		counts.TeamID == reported.TeamID &&
		counts.SigningID == reported.SigningID &&
		counts.CDHash == reported.CDHash &&
		counts.Compiler == reported.Compiler
}

func (counts RuleCounts) binaryWithoutTransitive() int32 {
	return counts.Binary - counts.Transitive
}
