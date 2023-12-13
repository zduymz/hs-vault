package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type SecretV1 struct {
	*Raw
}

func (s *SecretV1) Backup(ctx context.Context) error {
	s.L.Info("Start backup Secret V1", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup everything
	if err := s.RawBackup(ctx, keyPrefix, "/"); err != nil {
		return err
	}

	return nil
}

func (s *SecretV1) Restore(ctx context.Context) error {
	s.L.Info("Start restore Secret V1", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore everything
	if err := s.RawRestore(ctx, keyPrefix, "/"); err != nil {
		return err
	}

	return nil
}
