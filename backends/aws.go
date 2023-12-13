package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type AWS struct {
	*Raw
}

func (s *AWS) Backup(ctx context.Context) error {
	s.L.Info("Start backup AWS", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup everything
	if err := s.RawBackup(ctx, keyPrefix, "/"); err != nil {
		return err
	}

	return nil
}

func (s *AWS) Restore(ctx context.Context) error {
	s.L.Info("Start restore AWS", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore everything
	if err := s.RawRestore(ctx, keyPrefix, "/"); err != nil {
		return err
	}

	return nil
}
