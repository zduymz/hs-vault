package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type Database struct {
	*Raw
}

func (s *Database) Backup(ctx context.Context) error {
	s.L.Info("Start backup Database", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup config
	if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// backup role
	if err := s.RawBackup(ctx, keyPrefix, "role"); err != nil {
		return err
	}

	return nil
}

func (s *Database) Restore(ctx context.Context) error {
	s.L.Info("Start restore Database", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore config
	if err := s.RawRestore(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// restore roles
	if err := s.RawRestore(ctx, keyPrefix, "role"); err != nil {
		return err
	}
	return nil
}
