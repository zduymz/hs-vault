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
	t.L.Info("Start backup Transit", zap.String("path", t.Engine.Path))

	// Backup in normal mode
	paths, err := t.VaultWalk(ctx, t.Engine.Path, "keys")
	if err != nil {
		return err
	}

	for _, p := range paths {
		// enable exportable
		if _, err := t.Vault.Write(ctx, path.Join(t.Engine.Path, p, "config"), map[string]interface{}{
			"allow_plaintext_backup": true,
			"exportable":             true,
		}); err != nil {
			return err
		}

		// backup
		data, err := t.Vault.Read(ctx, path.Join(t.Engine.Path, "backup", path.Base(p)))
		if err != nil {
			return err
		}

		if err := t.WriteVaultResponse(ctx, p, data.Data); err != nil {
			return err
		}
	}

	return nil
}

func (t *Transit) Restore(ctx context.Context) error {
	t.L.Info("Start restore Transit", zap.String("path", t.Engine.Path))

	paths, err := t.LocalWalk(ctx, t.Options.RestorePath, "keys")
	if err != nil {
		return err
	}

	for _, p := range paths {
		data, err := t.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		payload := map[string]interface{}{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}

		if _, err := t.Vault.Write(ctx, path.Join(t.Engine.Path, "restore", path.Base(p)), payload); err != nil {
			return err
		}
	}

	return nil
}
