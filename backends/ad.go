package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"os"
	"path"
)

type AD struct {
	*Object
}

func (s *AD) Backup(ctx context.Context) error {
	s.L.Info("Start backup AD", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup config
	if s.Options.RawAccessible {
		if err := s.RawBackupSingleKey(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup roles
	return s.VaultBackupRoles(ctx, "roles")
}

func (s *AD) Restore(ctx context.Context) error {
	s.L.Info("Start restore AD", zap.String("path", s.Engine.Path))

	// restore config
	data, err := s.ReadFileAndB64Decode(ctx, "config")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		s.L.Warn("No config file found, skip restore AD configuration")
	} else {
		s.L.Info("Restore AD configuration")
		config := map[string]interface{}{}
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}

		payload := map[string]interface{}{}
		for k, v := range config["PasswordConf"].(map[string]interface{}) {
			payload[k] = v
		}
		for k, v := range config["ADConf"].(map[string]interface{}) {
			payload[k] = v
		}

		if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config"), payload); err != nil {
			return err
		}
	}

	return s.VaultRestoreRoles(ctx, "roles")
}
