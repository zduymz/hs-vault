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
	"path/filepath"
	"sort"
	"strings"
)

type SecretV2 struct {
	*Raw
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

func (s *SecretV2) vaultWalk(ctx context.Context, prefix string, start string) ([]string, error) {
	s.L.Debug("vaultWalk", zap.String("prefix", prefix), zap.String("start", start))
	var files []string

	resp, err := s.Vault.List(ctx, path.Join(prefix, start))
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			s.L.Debug("path is empty", zap.String("path", path.Join(prefix, start)))
			return nil, nil
		}
		return nil, err
	}

	keys, _ := resp.Data["keys"]
	for _, k := range keys.([]interface{}) {
		key := k.(string)
		if strings.HasSuffix(key, "/") {
			fs, err := s.vaultWalk(ctx, prefix, path.Join(start, key))
			if err != nil {
				return nil, err
			}
			files = append(files, fs...)
			continue
		}
		files = append(files, path.Join(start, key))
	}

	return files, nil
}

func (s *SecretV2) osWalk(ctx context.Context, prefix string, start string) ([]string, error) {
	s.L.Debug("osWalk", zap.String("prefix", prefix), zap.String("start", start))
	var output []string

	files, err := os.ReadDir(path.Join(prefix, start))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			fs, err := s.osWalk(ctx, prefix, filepath.Join(start, f.Name()))
			if err != nil {
				return nil, err
			}
			output = append(output, fs...)
			continue
		}
		output = append(output, filepath.Join(start, f.Name()))
	}

	return output, nil
}

func (s *SecretV2) getMetaData(ctx context.Context, k string) (*SecretV2Metadata, error) {
	s.L.Debug("getMetaData", zap.String("key", k))
	data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, "metadata", k))
	if err != nil {
		return nil, err
	}

	var meta SecretV2Metadata

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

// Backup single key
func (s *SecretV2) backupSingleKey(ctx context.Context, key string) error {
	s.L.Debug("backupSingleKey", zap.String("key", key))
	meta, err := s.getMetaData(ctx, key)
	if err != nil {
		return err
	}

	// it's redundant but just make sure key is ordered
	var versions []int
	for k, _ := range meta.Versions {
		//if v.Destroyed || v.DeletionTime != "" {
		//	continue
		//}
		versions = append(versions, k)
	}
	sort.Ints(versions)

	var v2Data = make(map[int]map[string]interface{})

	// collect all versions data
	for _, v := range versions {
		// skip destroyed or deleted version
		if meta.Versions[v].Destroyed || meta.Versions[v].DeletionTime != "" {
			v2Data[v] = map[string]interface{}{}
			continue
		}

		//options := vault.WithQueryParameters(url.Values{
		//	"version": {fmt.Sprintf("%d", v)},
		//})
		//data, err := s.Vault.Read(ctx, path.Join(s.Engine.Path, "data", key), options)
		//if err != nil {
		//	return err
		//}

		data, err := s.readData(ctx, key, v)
		if err != nil {
			return err
		}

		v2Data[v] = data
	}

	// create output format
	output := SecretV2Backup{
		*meta,
		v2Data,
	}

	content, err := json.Marshal(output)
	if err != nil {
		return err
	}

	b64content := base64.StdEncoding.EncodeToString(content)

	//create full path file if it's not existed
	oFile := path.Join(s.Options.BackupPath, key)
	if _, err := os.Stat(path.Dir(oFile)); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Dir(oFile), 0755); err != nil {
			return err
		}
	}

	f, err := os.Create(oFile)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(b64content); err != nil {
		return err
	}

	_ = f.Close()
	return nil
}

// Create a fake destroyed/deleted version
func (s *SecretV2) writeData(ctx context.Context, key string, data map[string]interface{}) error {
	s.L.Debug("writeData", zap.String("key", key))
	_, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, "data", key), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretV2) readData(ctx context.Context, key string, version int) (map[string]interface{}, error) {
	s.L.Debug("readData", zap.String("key", key), zap.Int("version", version))
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
	s.L.Debug("writeMetaData", zap.String("key", key))
	fmt.Println(metadata.CustomerMetadata)
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
	s.L.Debug("destroyVersions", zap.String("key", key), zap.Ints("versions", versions))
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

func (s *SecretV2) restoreSingleKey(ctx context.Context, key string) error {
	s.L.Debug("restoreSingleKey", zap.String("key", key))
	bs, err := os.ReadFile(path.Join(s.Options.RestorePath, key))
	if err != nil {
		return err
	}

	bd, err := base64.StdEncoding.DecodeString(string(bs))
	if err != nil {
		s.L.Error("base64 decode error", zap.Error(err))
		return err
	}

	var backup SecretV2Backup
	if err := json.Unmarshal(bd, &backup); err != nil {
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

		if err := s.writeData(ctx, key, data); err != nil {
			return err
		}

		if i == 1 {
			// update metadata
			if err := s.writeMetaData(ctx, key, &backup.MetaData); err != nil {
				return err
			}
		}
	}

	// destroy destroyed-mark versions
	if err := s.destroyVersions(ctx, key, destroyedVersions); err != nil {
		return err
	}
	return nil
}

func (s *SecretV2) Backup(ctx context.Context) error {
	s.L.Info("Start backup Secret V2", zap.String("path", s.Engine.Path))
	keyPrefix := path.Join(s.Engine.Path, "metadata")

	paths, err := s.vaultWalk(ctx, keyPrefix, "/")
	if err != nil {
		return err
	}

	for _, p := range paths {
		if err := s.backupSingleKey(ctx, p); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretV2) Restore(ctx context.Context) error {
	s.L.Info("Start restore Secret V2", zap.String("path", s.Engine.Path))
	files, err := s.osWalk(ctx, s.Options.RestorePath, "/")
	if err != nil {
		return nil
	}
	for _, file := range files {
		if err := s.restoreSingleKey(ctx, file); err != nil {
			return err
		}
	}

	return nil
}
