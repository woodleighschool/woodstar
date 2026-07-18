package protocol

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpx"
)

const (
	protobufContentType = "application/x-protobuf"
	maxRequestBodyBytes = 16 << 20
)

func (h handler) authorize(r *http.Request) error {
	token, ok := httpx.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		return errUnauthorized
	}

	ok, err := h.secretVerifier.Verify(r.Context(), agentauth.AgentSanta, token)
	if err != nil {
		return err
	}
	if !ok {
		return errUnauthorized
	}
	return nil
}

func validateTransportHeaders(r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != protobufContentType {
		return errUnsupportedMedia
	}
	if !strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		return errUnsupportedMedia
	}
	return nil
}

func decodeRequest(r *http.Request, msg proto.Message) error {
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	defer zr.Close()

	payload, err := io.ReadAll(io.LimitReader(zr, maxRequestBodyBytes+1))
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	if len(payload) > maxRequestBodyBytes {
		return errRequestBodyTooBig
	}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	return nil
}

func validateRequestMachineID(pathMachineID string, req machineIDProtoMessage) error {
	if req.GetMachineId() != pathMachineID {
		return fmt.Errorf(
			"%w: body machine_id %q does not match path machine_id %q",
			dbutil.ErrInvalidInput,
			req.GetMachineId(),
			pathMachineID,
		)
	}
	return nil
}

func writeProtoResponse(w http.ResponseWriter, msg proto.Message) error {
	payload, err := marshalCompressedProto(msg)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", protobufContentType)
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	return err
}

func marshalCompressedProto(msg proto.Message) ([]byte, error) {
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
