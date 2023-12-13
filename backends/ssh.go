package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type SSH struct {
	*Raw
}

func (s *SSH) Backup(ctx context.Context) error {
	s.L.Info("Start backup SSH", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup config
	if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// backup roles
	if err := s.RawBackup(ctx, keyPrefix, "roles"); err != nil {
		return err
	}

	return nil
}

func (s *SSH) Restore(ctx context.Context) error {
	s.L.Info("Start restore SSH", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore config
	if err := s.RawRestore(ctx, keyPrefix, "config"); err != nil {
		return err
	}

	// restore roles
	if err := s.RawRestore(ctx, keyPrefix, "roles"); err != nil {
		return err
	}
	return nil
}
