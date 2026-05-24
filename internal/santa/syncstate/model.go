package syncstate

type SyncType string

const (
	SyncTypeNormal SyncType = "normal"
	SyncTypeClean  SyncType = "clean"
)

type RuleCounts struct {
	Binary      int
	Certificate int
	TeamID      int
	SigningID   int
	CDHash      int
}
