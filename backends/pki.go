package backends

import (
	"context"
	"go.uber.org/zap"
	"path"
)

type PKI struct {
	*Object
}

func (s *PKI) Backup(ctx context.Context) error {
	s.L.With(zap.String("method", "Backup")).Debug("Start backup")
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup everything
	if err := s.RawBackup(ctx, keyPrefix, ""); err != nil {
		return err
	}

	return nil
}

func (s *PKI) Restore(ctx context.Context) error {
	s.L.With(zap.String("method", "Restore")).Debug("Start restore")
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore everything
	if err := s.RawRestoreSingleKey(ctx, keyPrefix, ""); err != nil {
		return err
	}

	return nil
}
