package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type AD struct {
	*Raw
}

func (s *AD) Backup(ctx context.Context) error {
	s.L.Info("Start backup AD", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup config
	if err := s.RawBackupSingleKey(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// backup cred
	if err := s.RawBackup(ctx, keyPrefix, "creds"); err != nil {
		return err
	}

	// backup roles
	if err := s.RawBackup(ctx, keyPrefix, "roles"); err != nil {
		return err
	}

	return nil
}

func (s *AD) Restore(ctx context.Context) error {
	s.L.Info("Start restore AD", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore config
	if err := s.RawRestoreSingleKey(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// restore config
	if err := s.RawRestore(ctx, keyPrefix, "creds"); err != nil {
		return err
	}

	// restore roles
	if err := s.RawRestore(ctx, keyPrefix, "roles"); err != nil {
		return err
	}
	return nil
}
