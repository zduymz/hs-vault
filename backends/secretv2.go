package backends

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/vault-client-go"
	"go.uber.org/zap"
	"net/url"
	"os"
	"path"
	"sort"
)

type SecretV2 struct {
	*Object
}

type SecretV2Metadata struct {
	Cas                bool                       `json:"cas_required"`
	MaxVersions        int                        `json:"max_versions"`
	Versions           map[int]SecretV2KeyVersion `json:"versions"`
	DeleteVersionAfter string                     `json:"delete_version_after"`
	CustomerMetadata   map[string]string          `json:"custom_metadata"`
	CurrentVersion     int                        `json:"current_version"`
}

type SecretV2KeyVersion struct {
	Destroyed    bool   `json:"destroyed"`
	DeletionTime string `json:"deletion_time"`
}

type SecretV2Backup struct {
	MetaData SecretV2Metadata               `json:"metadata"`
	Data     map[int]map[string]interface{} `json:"data"`
}

func (s *SecretV2) getMetaData(ctx context.Context, k string) (*SecretV2Metadata, error) {
	l := s.L.With(zap.String("method", "getMetaData"))

	vp := path.Join(s.Engine.Path, "metadata", k)
	l.Debug("vault read metadata", zap.String("path", vp))
	data, err := s.Vault.Read(ctx, vp)
	if err != nil {
		return nil, err
	}

	var meta SecretV2Metadata

	l.Debug("Marshal and Unmarshal data to SecretV2Metadata")
	// convert map[string]interface{} to json
	body, err := json.Marshal(data.Data)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// backupSingleKey return one key with all versions
func (s *SecretV2) backupSingleKey(ctx context.Context, key string) ([]byte, error) {
	l := s.L.With(zap.String("method", "backupSingleKey"))

	meta, err := s.getMetaData(ctx, key)
	if err != nil {
		return nil, err
	}

	// it's redundant but just make sure key is ordered
	var versions []int
	for k, _ := range meta.Versions {
		versions = append(versions, k)
	}
	sort.Ints(versions)

	var v2Data = make(map[int]map[string]interface{})

	// collect all versions data
	for _, v := range versions {
		l.Debug("Start check vault data", zap.String("key", key), zap.Int("version", v))
		// skip destroyed or deleted version
		if meta.Versions[v].Destroyed || meta.Versions[v].DeletionTime != "" {
			l.Debug("Skip destroyed or deleted version", zap.String("key", key), zap.Int("version", v))
			v2Data[v] = map[string]interface{}{}
			continue
		}

		l.Debug("Read data", zap.String("key", key), zap.Int("version", v))
		data, err := s.readData(ctx, key, v)
		if err != nil {
			return nil, err
		}

		v2Data[v] = data
	}

	// create output format
	output := SecretV2Backup{
		*meta,
		v2Data,
	}

	l.Debug("Marshal SecretV2Backup")
	content, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// Create a fake destroyed/deleted version
func (s *SecretV2) writeData(ctx context.Context, key string, data map[string]interface{}) error {
	s.L.With(zap.String("method", "writeData")).Debug("Write vault data", zap.String("key", key))
	_, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "data", key), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretV2) readData(ctx context.Context, key string, version int) (map[string]interface{}, error) {
	s.L.With(zap.String("method", "readData")).Debug("Read vault data", zap.String("key", key))
	options := vault.WithQueryParameters(url.Values{
		"version": {fmt.Sprintf("%d", version)},
	})
	data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, "data", key), options)
	if err != nil {
		return nil, err
	}

	return data.Data["data"].(map[string]interface{}), nil
}

// Write metadata
func (s *SecretV2) writeMetaData(ctx context.Context, key string, metadata *SecretV2Metadata) error {
	s.L.With(zap.String("method", "writeMetaData")).Debug("Write vault metadata", zap.String("key", key))
	if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "metadata", key), map[string]interface{}{
		"max_versions":         metadata.MaxVersions,
		"cas_required":         metadata.Cas,
		"delete_version_after": metadata.DeleteVersionAfter,
		"custom_metadata":      metadata.CustomerMetadata,
	}); err != nil {
		return err
	}

	return nil
}

// Destroy versions
func (s *SecretV2) destroyVersions(ctx context.Context, key string, versions []int) error {
	s.L.With(zap.String("method", "destroyVersions")).Debug("Destroy vault key versions", zap.String("key", key), zap.Ints("versions", versions))
	if len(versions) == 0 {
		return nil
	}

	if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "destroy", key), map[string]interface{}{
		"versions": versions,
	}); err != nil {
		return err
	}
	return nil
}

func (s *SecretV2) restoreSingleKey(ctx context.Context, key string, data []byte) error {
	l := s.L.With(zap.String("method", "restoreSingleKey"))

	l.Debug("Start restore key", zap.String("key", key))
	var backup SecretV2Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		s.L.Error("json unmarshal error", zap.Error(err))
		return err
	}

	var destroyedVersions []int
	for i := 1; i <= backup.MetaData.CurrentVersion; i++ {
		var data map[string]interface{}
		if _, ok := backup.MetaData.Versions[i]; !ok {
			destroyedVersions = append(destroyedVersions, i)
		} else if backup.MetaData.Versions[i].Destroyed || backup.MetaData.Versions[i].DeletionTime != "" {
			destroyedVersions = append(destroyedVersions, i)
		}

		if _, ok := backup.Data[i]; ok {
			data = backup.Data[i]
		}

		l.Debug("Restore version", zap.String("key", key), zap.Int("version", i))
		if err := s.writeData(ctx, key, data); err != nil {
			return err
		}

		if i == 1 {
			// update metadata after first version was created
			l.Debug("Restore metadata", zap.String("key", key))
			if err := s.writeMetaData(ctx, key, &backup.MetaData); err != nil {
				return err
			}
		}
	}

	// destroy destroyed-mark versions
	l.Debug("Start destroy versions")
	if err := s.destroyVersions(ctx, key, destroyedVersions); err != nil {
		return err
	}
	return nil
}

func (s *SecretV2) Backup(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Backup"))

	l.Debug("Back up metadata")
	keyPrefix := path.Join(s.Engine.Path, "metadata")
	paths, err := s.VaultWalk(ctx, keyPrefix, "/")
	if err != nil {
		return err
	}

	records := 100
	payload := make(map[string]string, records)
	count := 0
	for _, p := range paths {
		l.Debug("Start backup key", zap.String("key", p))
		bs, err := s.backupSingleKey(ctx, p)
		if err != nil {
			return err
		}
		payload[p] = base64.StdEncoding.EncodeToString(bs)
		if len(payload) == records {
			content, _ := json.Marshal(payload)
			of := fmt.Sprintf("file%d.json", count)
			l.Debug("Write a chunk of data to file", zap.String("file", of))
			if err := s.WriteData(ctx, of, content); err != nil {
				return err
			}
			payload = make(map[string]string, records)
			count += 1
		}

	}

	if len(payload) > 0 {
		content, _ := json.Marshal(payload)
		of := fmt.Sprintf("file%d.json", count)
		l.Debug("Write the last chunk of data to file", zap.String("file", of))
		if err := s.WriteData(ctx, of, content); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretV2) Restore(ctx context.Context) error {
	l := s.L.With(zap.String("method", "Backup"))

	l.Debug("Start restore")
	files, err := s.LocalWalk(ctx, s.Options.RestorePath, "/")
	if err != nil {
		return nil
	}
	for _, file := range files {
		l.Debug("Start restore file", zap.String("file", file))
		data, err := os.ReadFile(path.Join(s.Options.RestorePath, file))
		if err != nil {
			return err
		}

		entries := map[string]string{}
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}

		for k, v := range entries {
			bs, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}
			l.Debug("Run restore process", zap.String("key", k))
			if err := s.restoreSingleKey(ctx, k, bs); err != nil {
				return err
			}
		}

	}

	return nil
}
