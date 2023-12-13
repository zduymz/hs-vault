package backends

import (
	"context"
	"encoding/base64"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"go.uber.org/zap"
	"os"
	"path"
	"strings"
)

type Raw struct {
	Vault   *vault.Client
	Engine  *SecretEngine
	Options *Options
	L       *zap.Logger
}

func (r *Raw) RawBackupSingleKey(ctx context.Context, keyPrefix, key string) error {
	r.L.Debug("RawBackupSingleKey", zap.String("keyPrefix", keyPrefix), zap.String("key", key))
	rdata, err := r.Vault.System.RawRead(ctx, path.Join(keyPrefix, key))
	if err != nil {
		if strings.Contains(err.Error(), "being decompressed is empty") {
			return nil
		}
		return err
	}

	f, err := os.Create(path.Join(r.Options.BackupPath, key))
	if err != nil {
		return nil
	}

	value := rdata.Data.Value
	if r.Options.Base64Encode {
		value = base64.StdEncoding.EncodeToString([]byte(rdata.Data.Value))
	}

	if _, err = f.WriteString(value); err != nil {
		return err
	}
	_ = f.Close()
	return nil
}

// keyPrefix: logical/<uuid>
// key: config or roles/<role>
func (r *Raw) RawRestoreSingleKey(ctx context.Context, keyPrefix, key string) error {
	r.L.Debug("RawRestoreSingleKey", zap.String("keyPrefix", keyPrefix), zap.String("key", key))
	content, err := os.ReadFile(path.Join(r.Options.RestorePath, key))
	if err != nil {
		return err
	}

	if _, err = r.Vault.System.RawWrite(ctx, path.Join(keyPrefix, key), schema.RawWriteRequest{
		Encoding: "base64",
		Value:    string(content),
	}); err != nil {
		return nil
	}
	return nil
}

// keyPrefix: logical/<uuid>
// subKey: creds or roles
func (r *Raw) RawBackup(ctx context.Context, keyPrefix, subKey string) error {
	r.L.Debug("RawBackup", zap.String("keyPrefix", keyPrefix), zap.String("subKey", subKey))
	resp, err := r.Vault.System.RawList(ctx, path.Join(keyPrefix, subKey))
	if err != nil {
		return err
	}

	// create backup/*subKey* folder if not existed
	if _, err := os.Stat(path.Join(r.Options.BackupPath, subKey)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(r.Options.BackupPath, subKey), 0755); err != nil {
			return err
		}
	}

	for _, key := range resp.Data.Keys {
		// check if key is a folder
		if strings.HasSuffix(key, "/") {
			if err := r.RawBackup(ctx, keyPrefix, path.Join(subKey, key)); err != nil {
				return err
			}
			continue
		}

		if err := r.RawBackupSingleKey(ctx, keyPrefix, path.Join(subKey, key)); err != nil {
			return err
		}
	}

	return nil
}

func (r *Raw) RawRestore(ctx context.Context, keyPrefix, subKey string) error {
	files, err := os.ReadDir(path.Join(r.Options.RestorePath, subKey))
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			if err := r.RawRestore(ctx, keyPrefix, path.Join(subKey, file.Name())); err != nil {
				return err
			}
			continue
		}

		if err := r.RawRestoreSingleKey(ctx, keyPrefix, path.Join(subKey, file.Name())); err != nil {
			return err
		}
	}
	return nil
}
