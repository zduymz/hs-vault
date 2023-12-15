package backends

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"go.uber.org/zap"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Object struct {
	Vault   *vault.Client
	Engine  *SecretEngine
	Options *Options
	L       *zap.Logger
}

func (o *Object) RawBackupSingleKey(ctx context.Context, keyPrefix, key string) error {
	o.L.Debug("RawBackupSingleKey", zap.String("keyPrefix", keyPrefix), zap.String("key", key))
	rdata, err := o.Vault.System.RawRead(ctx, path.Join(keyPrefix, key))
	if err != nil {
		if strings.Contains(err.Error(), "being decompressed is empty") {
			return nil
		}
		return err
	}

	f, err := os.Create(path.Join(o.Options.BackupPath, key))
	if err != nil {
		return nil
	}

	value := rdata.Data.Value
	if o.Options.Base64Encode {
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
func (o *Object) RawRestoreSingleKey(ctx context.Context, keyPrefix, key string) error {
	o.L.Debug("RawRestoreSingleKey", zap.String("keyPrefix", keyPrefix), zap.String("key", key))
	content, err := os.ReadFile(path.Join(o.Options.RestorePath, key))
	if err != nil {
		return err
	}

	if _, err = o.Vault.System.RawWrite(ctx, path.Join(keyPrefix, key), schema.RawWriteRequest{
		Encoding: "base64",
		Value:    string(content),
	}); err != nil {
		return nil
	}
	return nil
}

// keyPrefix: logical/<uuid>
// subKey: creds or roles
func (o *Object) RawBackup(ctx context.Context, keyPrefix, subKey string) error {
	o.L.Debug("RawBackup", zap.String("keyPrefix", keyPrefix), zap.String("subKey", subKey))
	resp, err := o.Vault.System.RawList(ctx, path.Join(keyPrefix, subKey))
	if err != nil {
		return err
	}

	// create backup/*subKey* folder if not existed
	if _, err := os.Stat(path.Join(o.Options.BackupPath, subKey)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Join(o.Options.BackupPath, subKey), 0755); err != nil {
			return err
		}
	}

	for _, key := range resp.Data.Keys {
		// check if key is a folder
		if strings.HasSuffix(key, "/") {
			if err := o.RawBackup(ctx, keyPrefix, path.Join(subKey, key)); err != nil {
				return err
			}
			continue
		}

		if err := o.RawBackupSingleKey(ctx, keyPrefix, path.Join(subKey, key)); err != nil {
			return err
		}
	}

	return nil
}

func (o *Object) RawRestore(ctx context.Context, keyPrefix, subKey string) error {
	o.L.Debug("RawRestore", zap.String("keyPrefix", keyPrefix), zap.String("subKey", subKey))

	p := path.Join(o.Options.RestorePath, subKey)
	// check directory exist
	if _, err := os.Stat(p); os.IsNotExist(err) {
		o.L.Warn("path does not exist", zap.String("path", p))
		return nil
	}

	files, err := os.ReadDir(p)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			if err := o.RawRestore(ctx, keyPrefix, path.Join(subKey, file.Name())); err != nil {
				return err
			}
			continue
		}

		if err := o.RawRestoreSingleKey(ctx, keyPrefix, path.Join(subKey, file.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (o *Object) VaultWalk(ctx context.Context, prefix string, start string) ([]string, error) {
	o.L.Debug("vaultWalk", zap.String("prefix", prefix), zap.String("start", start))
	var files []string

	resp, err := o.Vault.List(ctx, path.Join(prefix, start))
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			o.L.Debug("path is empty", zap.String("path", path.Join(prefix, start)))
			return nil, nil
		}
		return nil, err
	}

	keys, _ := resp.Data["keys"]
	for _, k := range keys.([]interface{}) {
		key := k.(string)
		if strings.HasSuffix(key, "/") {
			fs, err := o.VaultWalk(ctx, prefix, path.Join(start, key))
			if err != nil {
				return nil, err
			}
			files = append(files, fs...)
			continue
		}
		files = append(files, path.Join(start, key))
	}

	return files, nil
}

func (o *Object) LocalWalk(ctx context.Context, prefix string, start string) ([]string, error) {
	o.L.Debug("LocalWalk", zap.String("prefix", prefix), zap.String("start", start))
	var output []string

	files, err := os.ReadDir(path.Join(prefix, start))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			fs, err := o.LocalWalk(ctx, prefix, filepath.Join(start, f.Name()))
			if err != nil {
				return nil, err
			}
			output = append(output, fs...)
			continue
		}
		output = append(output, filepath.Join(start, f.Name()))
	}

	return output, nil
}

func (o *Object) NormalBackup(ctx context.Context, key string) error {

	//return data.Data["data"].(map[string]interface{}), nil
	return nil
}

// ReadFileAndB64Decode read local file and return base64 decoded data
func (o *Object) ReadFileAndB64Decode(ctx context.Context, fp string) ([]byte, error) {
	bs, err := os.ReadFile(path.Join(o.Options.RestorePath, fp))
	if err != nil {
		return nil, err
	}

	bd, err := base64.StdEncoding.DecodeString(string(bs))
	if err != nil {
		o.L.Error("base64 decode error", zap.Error(err))
		return nil, err
	}

	return bd, nil
}

func (o *Object) WriteB64Data(ctx context.Context, fp string, content []byte) error {
	output := base64.StdEncoding.EncodeToString(content)
	of := path.Join(o.Options.BackupPath, fp)
	if _, err := os.Stat(path.Dir(of)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Dir(of), 0755); err != nil {
			return err
		}
	}

	f, err := os.Create(of)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(output); err != nil {
		return err
	}

	_ = f.Close()
	return nil
}

func (o *Object) WriteVaultResponse(ctx context.Context, fp string, data map[string]interface{}) error {
	content, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	return o.WriteB64Data(ctx, fp, content)
}

func (o *Object) VaultRestoreRoles(ctx context.Context, dir string) error {
	o.L.Info("Restore roles", zap.String("path", o.Engine.Path))
	paths, err := o.LocalWalk(ctx, o.Options.RestorePath, dir)
	if err != nil {
		return err
	}

	for _, p := range paths {
		data, err := o.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		payload := map[string]interface{}{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}

		if _, err := o.Vault.Write(ctx, path.Join(o.Engine.Path, p), payload); err != nil {
			return err
		}
	}
	return nil
}

func (o *Object) VaultBackupRoles(ctx context.Context, dir string) error {
	o.L.Info("Backup roles", zap.String("path", o.Engine.Path))
	paths, err := o.VaultWalk(ctx, o.Engine.Path, dir)
	if err != nil {
		return err
	}

	for _, p := range paths {
		data, err := o.Vault.Read(ctx, path.Join(o.Engine.Path, p))
		if err != nil {
			return err
		}

		if err := o.WriteVaultResponse(ctx, p, data.Data); err != nil {
			return err
		}

	}

	return nil
}
