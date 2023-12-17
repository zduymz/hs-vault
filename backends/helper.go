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
	l := o.L.With(zap.String("method", "RawBackupSingleKey"))

	vp := path.Join(keyPrefix, key)
	l.Debug("Read raw data", zap.String("path", vp))
	rdata, err := o.Vault.System.RawRead(ctx, vp)
	if err != nil {
		if strings.Contains(err.Error(), "being decompressed is empty") {
			return nil
		}
		return err
	}

	of := path.Join(o.Options.BackupPath, key)
	l.Debug("Create local file", zap.String("path", of))
	f, err := os.Create(of)
	if err != nil {
		return nil
	}

	value := base64.StdEncoding.EncodeToString([]byte(rdata.Data.Value))

	l.Debug("Write data to local file", zap.String("path", of))
	if _, err = f.WriteString(value); err != nil {
		return err
	}
	_ = f.Close()
	return nil
}

// keyPrefix: logical/<uuid>
// key: config or roles/<role>
func (o *Object) RawRestoreSingleKey(ctx context.Context, keyPrefix, key string) error {
	l := o.L.With(zap.String("method", "RawRestoreSingleKey"))

	f := path.Join(o.Options.RestorePath, key)
	l.Debug("Read local file", zap.String("path", f))
	content, err := os.ReadFile(f)
	if err != nil {
		return err
	}

	vp := path.Join(keyPrefix, key)
	l.Debug("Write raw data", zap.String("path", path.Join(keyPrefix, key)))
	if _, err = o.Vault.System.RawWrite(ctx, vp, schema.RawWriteRequest{
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
	l := o.L.With(zap.String("method", "RawBackup"))

	vp := path.Join(keyPrefix, subKey)
	l.Debug("List raw path", zap.String("path", vp))
	resp, err := o.Vault.System.RawList(ctx, vp)
	if err != nil {
		return err
	}

	//if _, err := os.Stat(path.Join(o.Options.BackupPath, subKey)); os.IsNotExist(err) {
	//	if err := os.MkdirAll(path.Join(o.Options.BackupPath, subKey), 0755); err != nil {
	//		return err
	//	}
	//}
	od := path.Join(o.Options.BackupPath, subKey)
	l.Debug("Create local directory", zap.String("path", od))
	if err := os.MkdirAll(od, 0755); err != nil {
		return err
	}

	for _, key := range resp.Data.Keys {
		l.Debug("Start backup process", zap.String("key", key))
		// check if key is a folder
		if strings.HasSuffix(key, "/") {
			l.Debug("key is folder, checking inside", zap.String("key", key))
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
	l := o.L.With(zap.String("method", "RawRestore"))

	p := path.Join(o.Options.RestorePath, subKey)
	l.Debug("Check restore path exist", zap.String("path", p))
	if _, err := os.Stat(p); os.IsNotExist(err) {
		l.Warn("path does not exist", zap.String("path", p))
		return nil
	}

	l.Debug("Read files from restore path", zap.String("path", p))
	files, err := os.ReadDir(p)
	if err != nil {
		return err
	}

	for _, file := range files {
		fp := path.Join(subKey, file.Name())
		if file.IsDir() {
			if err := o.RawRestore(ctx, keyPrefix, fp); err != nil {
				return err
			}
			continue
		}

		l.Debug("Start restore process", zap.String("file", fp))
		if err := o.RawRestoreSingleKey(ctx, keyPrefix, fp); err != nil {
			return err
		}
	}
	return nil
}

func (o *Object) VaultWalk(ctx context.Context, prefix string, start string) ([]string, error) {
	l := o.L.With(zap.String("method", "VaultWalk"))
	var files []string

	vp := path.Join(prefix, start)
	l.Debug("List vault path", zap.String("path", vp))
	resp, err := o.Vault.List(ctx, vp)
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			o.L.Debug("path is empty", zap.String("path", vp))
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
	l := o.L.With(zap.String("method", "LocalWalk"))
	var output []string

	lp := path.Join(prefix, start)
	l.Debug("List local path", zap.String("path", lp))
	files, err := os.ReadDir(lp)
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

// ReadFileAndB64Decode read local file and return base64 decoded data
func (o *Object) ReadFileAndB64Decode(ctx context.Context, f string) ([]byte, error) {
	l := o.L.With(zap.String("method", "ReadFileAndB64Decode"))

	fp := path.Join(o.Options.RestorePath, f)
	l.Debug("Read local file", zap.String("path", fp))
	bs, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	l.Debug("Decode base64 data", zap.String("path", fp))
	bd, err := base64.StdEncoding.DecodeString(string(bs))
	if err != nil {
		l.Error("decode error", zap.Error(err))
		return nil, err
	}

	return bd, nil
}

func (o *Object) WriteData(ctx context.Context, fp string, content []byte) error {
	l := o.L.With(zap.String("method", "WriteData"))

	of := path.Join(o.Options.BackupPath, fp)
	l.Debug("Create parent directory if needed", zap.String("path", of))
	if err := os.MkdirAll(path.Dir(of), 0755); err != nil {
		l.Error("create error", zap.Error(err))
		return err
	}

	l.Debug("Create local file", zap.String("path", of))
	f, err := os.Create(of)
	if err != nil {
		return err
	}

	l.Debug("Write data to local file", zap.String("path", of))
	if _, err = f.Write(content); err != nil {
		return err
	}

	_ = f.Close()
	return nil
}

func (o *Object) WriteB64Data(ctx context.Context, fp string, content []byte) error {
	l := o.L.With(zap.String("method", "WriteB64Data"))
	l.Debug("Encode base64 data", zap.String("path", fp))
	output := base64.StdEncoding.EncodeToString(content)
	return o.WriteData(ctx, fp, []byte(output))
}

func (o *Object) WriteVaultResponse(ctx context.Context, fp string, data map[string]interface{}) error {
	l := o.L.With(zap.String("method", "WriteVaultResponse"))
	l.Debug("Marshal data", zap.String("path", fp))
	content, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	return o.WriteB64Data(ctx, fp, content)
}

func (o *Object) VaultRestoreRoles(ctx context.Context, dir string) error {
	l := o.L.With(zap.String("method", "VaultRestoreRoles"))

	l.Debug("List all local files", zap.String("path", path.Join(o.Options.RestorePath, dir)))
	paths, err := o.LocalWalk(ctx, o.Options.RestorePath, dir)
	if err != nil {
		return err
	}

	for _, p := range paths {
		l.Debug("Read local file and decode base64", zap.String("path", p))
		data, err := o.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		payload := map[string]interface{}{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}

		vp := path.Join(o.Engine.Path, p)
		l.Debug("Write data to vault", zap.String("path", vp))
		if _, err := o.Vault.Write(ctx, vp, payload); err != nil {
			return err
		}
	}
	return nil
}

func (o *Object) VaultBackupRoles(ctx context.Context, dir string) error {
	l := o.L.With(zap.String("method", "VaultRestoreRoles"))

	l.Debug("List all vault paths", zap.String("path", path.Join(o.Engine.Path, dir)))
	paths, err := o.VaultWalk(ctx, o.Engine.Path, dir)
	if err != nil {
		return err
	}

	for _, p := range paths {
		vp := path.Join(o.Engine.Path, p)

		l.Debug("Read data from vault", zap.String("path", vp))
		data, err := o.Vault.Read(ctx, vp)
		if err != nil {
			return err
		}

		l.Debug("Process vault response")
		if err := o.WriteVaultResponse(ctx, p, data.Data); err != nil {
			return err
		}

	}

	return nil
}
