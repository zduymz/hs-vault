package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"path"
)

type Transit struct {
	*Object
}

func (t *Transit) Backup(ctx context.Context) error {
	l := t.L.With(zap.String("method", "Backup"))

	// Backup in normal mode
	l.Debug("Start backup keys")
	paths, err := t.VaultWalk(ctx, t.Engine.Path, "keys")
	if err != nil {
		return err
	}

	for _, p := range paths {
		vp := path.Join(t.Engine.Path, p, "config")
		l.Debug("Enable exportable for key", zap.String("path", vp))
		if _, err := t.Vault.Write(ctx, vp, map[string]interface{}{
			"allow_plaintext_backup": true,
			"exportable":             true,
		}); err != nil {
			return err
		}

		// backup
		bk := path.Join(t.Engine.Path, "backup", path.Base(p))
		l.Debug("Read backup key endpoint", zap.String("path", bk))
		data, err := t.Vault.Read(ctx, bk)
		if err != nil {
			return err
		}

		l.Debug("Write vault response to local file")
		if err := t.WriteVaultResponse(ctx, p, data.Data); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transit) Restore(ctx context.Context) error {
	l := t.L.With(zap.String("method", "Restore"))

	l.Debug("Start restore keys")
	paths, err := t.LocalWalk(ctx, t.Options.RestorePath, "keys")
	if err != nil {
		return err
	}

	for _, p := range paths {
		l.Debug("Read and decode local file", zap.String("path", p))
		data, err := t.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		l.Debug("Unmarshal data")
		payload := map[string]interface{}{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}

		vp := path.Join(t.Engine.Path, "restore", path.Base(p))
		l.Debug("Write data to vault", zap.String("path", vp))
		if _, err := t.Vault.Write(ctx, vp, payload); err != nil {
			return err
		}
	}

	return nil
}
