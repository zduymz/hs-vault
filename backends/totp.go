package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type TOTP struct {
	*Raw
}

func (t *TOTP) Backup(ctx context.Context) error {
	t.L.Info("Start backup TOTP", zap.String("path", t.Engine.Path))
	keyPrefix := path.Join("logical", t.Engine.UUID)
	if err := t.RawBackup(ctx, keyPrefix, "key"); err != nil {
		return err
	}

	return nil
}

func (t *TOTP) Restore(ctx context.Context) error {
	t.L.Info("Start restore TOTP", zap.String("path", t.Engine.Path))
	keyPrefix := path.Join("logical", t.Engine.UUID)
	if err := t.RawRestore(ctx, keyPrefix, "key"); err != nil {
		return err
	}

	return nil
}
