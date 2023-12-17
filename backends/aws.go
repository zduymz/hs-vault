package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"os"
	"path"
)

type AWS struct {
	*Object
}

func (s *AWS) Backup(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Backup"))

	if s.Options.RawAccessible {
		l.Debug("Start backup configuration")
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	l.Debug("Start backup lease time")
	data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, "config/lease"))
	if err != nil {
		return err
	}

	if err := s.WriteVaultResponse(ctx, "config/lease", data.Data); err != nil {
		return err
	}

	s.L.Debug("Start backup roles")
	return s.VaultBackupRoles(ctx, "roles")

}

func (s *AWS) Restore(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Restore"))

	l.Debug("Start restore root configuration")
	if data, err := s.ReadFileAndB64Decode(ctx, "config/root"); err != nil {
		if os.IsNotExist(err) {
			s.L.Warn("Root configuration not found, skip restore root configuration")
		} else {
			payload := map[string]interface{}{}
			l.Debug("Unmarshal root configuration")
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
			l.Debug("Write root configuration to vault")
			if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config/root"), payload); err != nil {
				return err
			}
		}
	}

	l.Debug("Start restore lease configuration")
	if data, err := s.ReadFileAndB64Decode(ctx, "config/lease"); err != nil {
		if os.IsNotExist(err) {
			s.L.Warn("Root configuration not found, skip restore root configuration")
		} else {
			payload := map[string]interface{}{}
			l.Debug("Unmarshal lease configuration")
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
			l.Debug("Write lease configuration to vault")
			if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config/lease"), payload); err != nil {
				return err
			}
		}
	}

	l.Debug("Start restore roles")
	return s.VaultRestoreRoles(ctx, "roles")
}
