package protocol

import (
	"fmt"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func postflightRequestFromProto(req *syncv1.PostflightRequest) (santa.PostflightRequest, error) {
	syncType, err := syncTypeFromProto(req.GetSyncType())
	if err != nil {
		return santa.PostflightRequest{}, err
	}
	return santa.PostflightRequest{
		RulesReceived:  int32(req.GetRulesReceived()),
		RulesProcessed: int32(req.GetRulesProcessed()),
		SyncType:       syncType,
		RulesHash:      req.GetRulesHash(),
	}, nil
}

func postflightResponseToProto(santa.PostflightResponse) (*syncv1.PostflightResponse, error) {
	return &syncv1.PostflightResponse{}, nil
}

func syncTypeFromProto(value syncv1.SyncType) (syncstate.SyncType, error) {
	switch value {
	case syncv1.SyncType_NORMAL:
		return syncstate.SyncTypeNormal, nil
	case syncv1.SyncType_CLEAN:
		return syncstate.SyncTypeClean, nil
	case syncv1.SyncType_CLEAN_ALL:
		return syncstate.SyncTypeCleanAll, nil
	case syncv1.SyncType_SYNC_TYPE_UNSPECIFIED,
		syncv1.SyncType_CLEAN_STANDALONE,
		syncv1.SyncType_CLEAN_RULES,
		syncv1.SyncType_CLEAN_FILE_ACCESS_RULES:
	default:
	}
	return "", fmt.Errorf("%w: unsupported sync_type %q", dbutil.ErrInvalidInput, value)
}
