package objectupload

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const Label = "munki upload"

// IconObjectPrefix namespaces software icon objects in storage.
const IconObjectPrefix = "munki/icons"

func Create(
	ctx context.Context,
	objects *storage.ObjectStore,
	presigner storage.Presigner,
	prefix string,
	filename string,
	contentType string,
) (*storage.Object, storage.UploadTarget, error) {
	obj, err := objects.CreatePending(ctx, prefix, filename, contentType)
	if err != nil {
		return nil, storage.UploadTarget{}, err
	}

	target, err := presigner.PresignPut(ctx, obj.Key(), 0, storage.PutOptions{ContentType: contentType})
	if err != nil {
		return nil, storage.UploadTarget{}, err
	}
	return obj, target, nil
}

func Confirm(
	ctx context.Context,
	objects *storage.ObjectStore,
	prefix string,
	objectID int64,
	attach func(objectID int64) error,
) (*storage.Object, error) {
	obj, err := objects.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if obj.Prefix != prefix {
		return nil, fmt.Errorf("%w: object has the wrong Munki prefix", dbutil.ErrInvalidInput)
	}
	confirmed, err := objects.ConfirmUploaded(ctx, objectID)
	if errors.Is(err, storage.ErrObjectNotFound) {
		return nil, fmt.Errorf("%w: uploaded object does not exist", dbutil.ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	if err := attach(confirmed.ID); err != nil {
		_ = objects.DeleteUnreferenced(ctx, confirmed.ID)
		return nil, err
	}
	return confirmed, nil
}

func ContentURL(ctx context.Context, presigner storage.Presigner, o storage.Object) (string, error) {
	return presigner.PresignGet(ctx, o.Key(), 0, storage.GetOptions{ContentType: o.ContentType})
}
