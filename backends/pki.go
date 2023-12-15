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
	s.L.Info("Start backup PKI", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// backup everything
	if err := s.RawBackup(ctx, keyPrefix, ""); err != nil {
		return err
	}

	return nil
}

func (s *PKI) Restore(ctx context.Context) error {
	s.L.Info("Start restore PKI", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join("logical", s.Engine.UUID)

	// restore everything
	if err := s.RawRestoreSingleKey(ctx, keyPrefix, ""); err != nil {
		return err
	}

	return nil
}
