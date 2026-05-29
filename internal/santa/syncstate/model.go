package syncstate

type SyncType string

const (
	SyncTypeNormal SyncType = "normal"
	SyncTypeClean  SyncType = "clean"
)

type RuleCounts struct {
	Binary      int32
	Certificate int32
	TeamID      int32
	SigningID   int32
	CDHash      int32
}
