package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"os"
	"path"
)

type Database struct {
	*Object
}

//type DatabaseRole struct {
//	DBName               string                 `json:"db_name"`
//	DefaultTTL           int                    `json:"default_ttl"`
//	MaxTTL               int                    `json:"max_ttl"`
//	CreationStatements   []string               `json:"creation_statements"`
//	RevocationStatements []string               `json:"revocation_statements"`
//	RollbackStatements   []string               `json:"rollback_statements"`
//	RenewStatements      []string               `json:"renew_statements"`
//	CredentialType       string                 `json:"credential_type"`
//	CredentialConfig     map[string]interface{} `json:"credential_config"`
//}

type DatabaseConfig struct {
	AllowedRoles      []string               `json:"allowed_roles"`
	ConnectionDetails map[string]interface{} `json:"connection_details"`
	PasswordPolicy    string                 `json:"password_policy"`
	PluginName        string                 `json:"plugin_name"`
	PluginVersion     string                 `json:"plugin_version"`
	RootRotationStat  string                 `json:"root_rotation_statements"`
	VerfiyConnection  bool                   `json:"verify_connection"`
}

//type MySQL struct {
//	DatabaseConfig
//	ConnectionURL      string `json:"connection_details.connection_url"`
//	Username           string `json:"connection_details.username"`
//	Password           string `json:"connection_details.password"`
//	MaxOpenConns       int    `json:"connection_details.max_open_connections"`
//	MaxIdleConns       int    `json:"connection_details.max_idle_connections"`
//	MaxLifetime        int    `json:"connection_details.max_connection_lifetime"`
//	AuthType           string `json:"connection_details.auth_type"`
//	ServiceAccountJson string `json:"connection_details.service_account_json"`
//	UserNameTemplate   string `json:"connection_details.username_template"`
//	DisableEscaping    bool   `json:"connection_details.disable_escaping"`
//	TLSCertKey         string `json:"connection_details.tls_certificate_key"`
//	TLSCA              string `json:"connection_details.tls_ca"`
//	TLSServerName      string `json:"connection_details.tls_server_name"`
//	TLSSkipVerify      bool   `json:"connection_details.tls_skip_verify"`
//}
//
//type Postgres struct {
//	DatabaseConfig
//	ConnectionURL          string `json:"connection_details.connection_url"`
//	Username               string `json:"connection_details.username"`
//	Password               string `json:"connection_details.password"`
//	MaxOpenConns           int    `json:"connection_details.max_open_connections"`
//	MaxIdleConns           int    `json:"connection_details.max_idle_connections"`
//	MaxLifetime            int    `json:"connection_details.max_connection_lifetime"`
//	AuthType               string `json:"connection_details.auth_type"`
//	ServiceAccountJson     string `json:"connection_details.connection_details.service_account_json"`
//	UserNameTemplate       string `json:"connection_details.username_template"`
//	DisableEscaping        bool   `json:"connection_details.disable_escaping"`
//	PasswordAuthentication string `json:"connection_details.password_authentication"`
//}

func (s *Database) Backup(ctx context.Context) error {
	s.L.Info("Start backup Database", zap.String("path", s.Engine.Path))

	// config backup only support raw mode
	if s.Options.RawAccessible {
		keyPrefix := path.Join("logical", s.Engine.UUID)
		if err := s.RawBackup(ctx, keyPrefix, "config"); err != nil {
			return err
		}
	}

	// backup role
	return s.VaultBackupRoles(ctx, "roles")
}

func (s *Database) Restore(ctx context.Context) error {
	s.L.Info("Start restore Database", zap.String("path", s.Engine.Path))

	// restore config
	var paths []string
	paths, err := s.LocalWalk(ctx, s.Options.RestorePath, "config")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.IsNotExist(err) {
		s.L.Warn("No config directory found, skip restore Database configuration")
	}

	for _, p := range paths {
		data, err := s.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		var d DatabaseConfig
		if err := json.Unmarshal(data, &d); err != nil {
			return err
		}

		payload := map[string]interface{}{}
		payload["allowed_roles"] = d.AllowedRoles
		payload["password_policy"] = d.PasswordPolicy
		payload["plugin_name"] = d.PluginName
		payload["plugin_version"] = d.PluginVersion
		payload["root_rotation_statements"] = d.RootRotationStat
		payload["verify_connection"] = d.VerfiyConnection
		for k, v := range d.ConnectionDetails {
			payload[k] = v
		}

		//fmt.Printf("%+v\n", payload)
		if _, err := s.Vault.Write(ctx, path.Join(s.Engine.Path, p), payload); err != nil {
			return err
		}
	}

	// restore role
	return s.VaultRestoreRoles(ctx, "roles")
}
