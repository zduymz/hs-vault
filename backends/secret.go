package backends

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path"
)

type SecretV1 struct {
	*Object
}

func (s *SecretV1) Backup(ctx context.Context) error {
	s.L.Info("Start backup Secret V1", zap.String("path", s.Engine.Path))

	// Backup in normal mode
	paths, err := s.VaultWalk(ctx, s.Engine.Path, "/")
	if err != nil {
		return err
	}

	records := 100
	count := 0
	payload := make(map[string]string, records)

	for _, p := range paths {
		data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, p))
		if err != nil {
			return err
		}

		bs, err := json.Marshal(data.Data)
		if err != nil {
			return err
		}

		payload[p] = base64.StdEncoding.EncodeToString(bs)
		if len(payload) >= records {
			content, _ := json.Marshal(payload)
			if err := s.WriteData(ctx, fmt.Sprintf("file%d", count), content); err != nil {
				return err
			}
			payload = make(map[string]string, records)
			count += 1
		}
	}

	return nil
}

func (s *SecretV1) Restore(ctx context.Context) error {
	s.L.Info("Start restore Secret V1", zap.String("path", s.Engine.Path))

	files, err := s.LocalWalk(ctx, s.Options.RestorePath, "/")
	if err != nil {
		return err
	}

	for _, f := range files {
		data, err := os.ReadFile(path.Join(s.Options.RestorePath, f))
		if err != nil {
			return err
		}

		entries := map[string]string{}
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}

		for key, b64value := range entries {
			bs, err := base64.StdEncoding.DecodeString(b64value)
			if err != nil {
				return err
			}

			var value map[string]interface{}
			if err := json.Unmarshal(bs, &value); err != nil {
				return err
			}

			s.L.Debug("Restore key", zap.String("key", key))
			if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, key), value); err != nil {
				return err
			}
		}
	}

	return nil
}
