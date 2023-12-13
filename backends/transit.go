package backends

import (
	"context"
	"go.uber.org/zap"
	"os"
	"path"
	"strings"
)

type Transit struct {
	*Raw
}

func (t *Transit) Backup(ctx context.Context) error {
	t.L.Info("Start backup Transit", zap.String("path", t.Engine.Path))
	resp, err := t.Vault.List(ctx, path.Join(t.Engine.Path, "keys"))
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			t.L.Debug("path is empty", zap.String("path", path.Join(t.Engine.Path, "keys")))
			return nil
		}
		return err
	}

	for _, key := range resp.Data["keys"].([]interface{}) {
		keyName := key.(string)
		keyConfig := path.Join(t.Engine.Path, "keys", keyName, "config")

		// enable exportable
		if _, err := t.Vault.Write(ctx, keyConfig, map[string]interface{}{
			"allow_plaintext_backup": true,
			"exportable":             true,
		}); err != nil {
			return err
		}

		// backup
		resp, err := t.Vault.Read(ctx, t.Engine.Path+"/backup/"+keyName)
		if err != nil {
			return err
		}

		// save to file
		f, err := os.Create(path.Join(t.Options.BackupPath, keyName))
		if err != nil {
			return err
		}

		if _, err = f.WriteString(resp.Data["backup"].(string)); err != nil {
			return err
		}

		_ = f.Close()
	}

	return nil
}

func (t *Transit) Restore(ctx context.Context) error {
	t.L.Info("Start restore Transit", zap.String("path", t.Engine.Path))
	files, err := os.ReadDir(t.Options.RestorePath)
	if err != nil {
		return nil
	}
	for _, file := range files {
		// if directory found, ignore
		if file.IsDir() {
			t.L.Info("Skip directory", zap.String("name", file.Name()))
			continue
		}

		// read file content
		t.L.Debug("Transit read backup file", zap.String("name", file.Name()))
		content, err := os.ReadFile(path.Join(t.Options.RestorePath, file.Name()))
		if err != nil {
			return err
		}

		// restore
		t.L.Debug("Transit Restore", zap.String("name", path.Join(t.Engine.Path, "restore", file.Name())))
		if _, err := t.Vault.Write(ctx, path.Join(t.Engine.Path, "restore", file.Name()), map[string]interface{}{
			"backup": string(content),
		}); err != nil {
			return err
		}
	}

	return nil
}
