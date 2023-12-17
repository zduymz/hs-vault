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
	l := s.L.With(zap.String("method", "Restore"))

	// backup config
	if s.Options.RawAccessible {
		l.Debug("Start backup config")
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackupSingleKey(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup roles
	l.Debug("Start backup roles")
	return s.VaultBackupRoles(ctx, "roles")
}

func (s *AD) Restore(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Restore"))

	l.Debug("Start restore config")
	l.Debug("Read and decode config file")
	data, err := s.ReadFileAndB64Decode(ctx, "config")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		l.Warn("No config file found, skip restore AD configuration")
	} else {
		l.Info("Restore AD configuration")
		config := map[string]interface{}{}
		l.Debug("Unmarshal config file")
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

		l.Debug("Write config to Vault")
		if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config"), payload); err != nil {
			return err
		}
	}

	l.Debug("Start restore roles")
	return s.VaultRestoreRoles(ctx, "roles")
}
