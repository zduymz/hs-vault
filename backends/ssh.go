package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"os"
	"path"
)

type SSH struct {
	*Object
}

func (s *SSH) Backup(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Backup"))

	// backup config
	if s.Options.RawAccessible {
		l.Debug("Start backup config")
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup roles
	l.Debug("Start backup roles")
	return s.VaultBackupRoles(ctx, "roles")
}

func (s *SSH) Restore(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Restore"))

	// restore config
	CAPublicKeyF := path.Join("config", "ca_public_key")
	CAPrivateKeyF := path.Join("config", "ca_private_key")
	var CAPublicKey, CAPrivateKey map[string]interface{}

	l.Debug("Read and decode ca public key")
	tmpCAPublicKey, err := s.ReadFileAndB64Decode(ctx, CAPublicKeyF)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	l.Debug("Read and decode ca private key")
	tmpCAPrivateKey, err := s.ReadFileAndB64Decode(ctx, CAPrivateKeyF)
	if err != nil != os.IsNotExist(err) {
		return err
	}

	if tmpCAPrivateKey == nil || tmpCAPublicKey == nil {
		l.Warn("Skip restore SSH config, because of missing CA keys")
	} else {
		l.Info("Start restore SSH config")
		if err := json.Unmarshal(tmpCAPublicKey, &CAPublicKey); err != nil {
			return err
		}
		if err := json.Unmarshal(tmpCAPrivateKey, &CAPrivateKey); err != nil {
			return err
		}

		if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config/ca"), map[string]interface{}{
			"private_key":          CAPrivateKey["key"],
			"public_key":           CAPublicKey["key"],
			"generate_signing_key": false,
		}); err != nil {
			return err
		}
	}

	// restore roles
	l.Info("Start restore SSH roles")
	return s.VaultRestoreRoles(ctx, "roles")
}
