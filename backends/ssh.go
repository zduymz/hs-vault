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
	s.L.Info("Start backup SSH", zap.String("path", s.Engine.Path))

	// backup config
	if s.Options.RawAccessible {
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup roles
	return s.VaultBackupRoles(ctx, "roles")
}

func (s *SSH) Restore(ctx context.Context) error {
	s.L.Info("Start restore SSH", zap.String("path", s.Engine.Path))

	// restore config
	CAPublicKeyF := path.Join("config", "ca_public_key")
	CAPrivateKeyF := path.Join("config", "ca_private_key")
	var CAPublicKey, CAPrivateKey map[string]interface{}

	tmpCAPublicKey, err := s.ReadFileAndB64Decode(ctx, CAPublicKeyF)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	tmpCAPrivateKey, err := s.ReadFileAndB64Decode(ctx, CAPrivateKeyF)
	if err != nil != os.IsNotExist(err) {
		return err
	}

	if tmpCAPrivateKey == nil || tmpCAPublicKey == nil {
		s.L.Warn("Skip restore SSH config, because of missing CA keys")
	} else {
		s.L.Info("Restore SSH config")
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
	return s.VaultRestoreRoles(ctx, "roles")
}
