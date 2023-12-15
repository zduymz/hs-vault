package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"path"
)

type SecretV1 struct {
	*Object
}

func (s *SecretV1) Backup(ctx context.Context) error {
	s.L.Info("Start backup Secret V1", zap.String("path", s.Engine.Path))

	//if s.Options.RawAccessible {
	//	keyPrefix := path.Join("logical", s.Engine.UUID)
	//	return s.RawBackup(ctx, keyPrefix, "/")
	//}

	// Backup in normal mode
	paths, err := s.VaultWalk(ctx, s.Engine.Path, "/")
	if err != nil {
		return err
	}

	for _, p := range paths {
		data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, p))
		if err != nil {
			return err
		}

		if err := s.WriteVaultResponse(ctx, p, data.Data); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretV1) Restore(ctx context.Context) error {
	s.L.Info("Start restore Secret V1", zap.String("path", s.Engine.Path))
	//if s.Options.RawAccessible {
	//	keyPrefix := path.Join("logical", s.Engine.UUID)
	//	return s.RawRestore(ctx, keyPrefix, "/")
	//}

	// Restore in normal mode
	paths, err := s.LocalWalk(ctx, s.Options.RestorePath, "/")
	if err != nil {
		return err
	}

	for _, p := range paths {
		data, err := s.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		d := map[string]interface{}{}
		if err := json.Unmarshal(data, &d); err != nil {
			return err
		}

		if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, p), d); err != nil {
			return err
		}
	}

	return nil
}
