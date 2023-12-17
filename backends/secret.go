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
	l := s.L.With(zap.String("method", "Backup"))

	l.Debug("Start backup")
	paths, err := s.VaultWalk(ctx, s.Engine.Path, "/")
	if err != nil {
		return err
	}

	records := 100
	count := 0
	payload := make(map[string]string, records)

	for _, p := range paths {
		vp := path.Join(s.Engine.Path, p)
		l.Debug("Backup key", zap.String("path", vp))
		data, err := s.Vault.Read(ctx, vp)
		if err != nil {
			return err
		}

		l.Debug("Marshal data from local file")
		bs, err := json.Marshal(data.Data)
		if err != nil {
			return err
		}

		payload[p] = base64.StdEncoding.EncodeToString(bs)
		if len(payload) >= records {
			l.Debug("Marshal data from map value")
			content, _ := json.Marshal(payload)

			of := fmt.Sprintf("file%d.json", count)
			l.Debug("Write a chunk data to file", zap.String("file", of))
			if err := s.WriteData(ctx, of, content); err != nil {
				return err
			}
			payload = make(map[string]string, records)
			count += 1
		}
	}

	if len(payload) > 0 {
		l.Debug("Marshal last block data from map value")
		content, _ := json.Marshal(payload)
		of := fmt.Sprintf("file%d.json", count)
		l.Debug("Write last chunk data to file", zap.String("file", of))
		if err := s.WriteData(ctx, of, content); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretV1) Restore(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Restore"))

	l.Debug("Start restore")
	files, err := s.LocalWalk(ctx, s.Options.RestorePath, "/")
	if err != nil {
		return err
	}

	for _, f := range files {
		fp := path.Join(s.Options.RestorePath, f)
		l.Debug("Read local file", zap.String("file", fp))
		data, err := os.ReadFile(fp)
		if err != nil {
			return err
		}

		l.Debug("Unmarshal data from local file")
		entries := map[string]string{}
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}

		for key, b64value := range entries {
			l.Debug("Decode base64 entry value", zap.String("key", key), zap.String("value", b64value))
			bs, err := base64.StdEncoding.DecodeString(b64value)
			if err != nil {
				return err
			}

			l.Debug("Unmarshal data from key entry", zap.String("key", key))
			var value map[string]interface{}
			if err := json.Unmarshal(bs, &value); err != nil {
				return err
			}

			vp := path.Join(s.Engine.Path, key)
			l.Debug("Write data to vault key", zap.String("key", vp))
			if _, err := s.Vault.Write(ctx, vp, value); err != nil {
				return err
			}
		}
	}

	return nil
}
