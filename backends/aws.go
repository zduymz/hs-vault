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
	s.L.Info("Start backup AWS", zap.String("path", s.Engine.Path))

	if s.Options.RawAccessible {
		s.L.Debug("Backup root configuration")
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup lease time
	s.L.Debug("Backup lease configuration")
	data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, "config/lease"))
	if err != nil {
		return err
	}

	if err := s.WriteVaultResponse(ctx, "config/lease", data.Data); err != nil {
		return err
	}

	// backup role
	return s.VaultBackupRoles(ctx, "roles")
	//s.L.Debug("Backup roles")
	//paths, err := s.VaultWalk(ctx, s.Engine.Path, "roles")
	//if err != nil {
	//	return err
	//}
	//
	//for _, p := range paths {
	//	data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, p))
	//	if err != nil {
	//		return err
	//	}
	//
	//	if err := s.WriteVaultResponse(ctx, p, data.Data); err != nil {
	//		return err
	//	}
	//
	//}
	//
	//return nil
}

func (s *AWS) Restore(ctx context.Context) error {
	s.L.Info("Start restore AWS", zap.String("path", s.Engine.Path))

	// restore root configuration
	if data, err := s.ReadFileAndB64Decode(ctx, "config/root"); err != nil {
		if os.IsNotExist(err) {
			s.L.Warn("Root configuration not found, skip restore root configuration")
		} else {
			payload := map[string]interface{}{}
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
			if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config/root"), payload); err != nil {
				return err
			}
		}
	}

	// restore lease configuration
	if data, err := s.ReadFileAndB64Decode(ctx, "config/lease"); err != nil {
		if os.IsNotExist(err) {
			s.L.Warn("Root configuration not found, skip restore root configuration")
		} else {
			payload := map[string]interface{}{}
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
			if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "config/lease"), payload); err != nil {
				return err
			}
		}
	}

	// restore roles
	return s.VaultRestoreRoles(ctx, "roles")

	//paths, err := s.LocalWalk(ctx, s.Options.RestorePath, "roles")
	//if err != nil {
	//	return err
	//}
	//
	//for _, p := range paths {
	//	data, err := s.ReadFileAndB64Decode(ctx, p)
	//	if err != nil {
	//		return err
	//	}
	//
	//	payload := map[string]interface{}{}
	//	if err := json.Unmarshal(data, &payload); err != nil {
	//		return err
	//	}
	//
	//	if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, p), payload); err != nil {
	//		return err
	//	}
	//}
	//
	//return nil
}
