package backends

import (
	"context"
	"github.com/hashicorp/vault-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"path"
	"strings"
)

type SecretEngine struct {
	Path string
	Type string
	UUID string
}

type SecretEngineResponse struct {
	Accessor    string
	Description string
	Local       bool
	Type        string
	Uuid        string
}

// Options
// BackupPath is directory where all backup engines should be stored, default is backup/<engine path>.<engine type>/
// RestorePath could be directory contains all backup engines, eg: backup/
//			or directory engine itself when restore specific engine, eg: backup/<engine path>.<engine type>/

type Options struct {
	Base64Encode   bool
	CompressedFile bool
	BackupPath     string
	RestorePath    string
	LogLevel       string
}

type Engine interface {
	Backup(context.Context) error
	Restore(context.Context) error
}

type EngineType string

const (
	RawEngine      EngineType = "raw"
	SSHEngine      EngineType = "ssh"
	TOTPEngine     EngineType = "totp"
	ADEngine       EngineType = "ad"
	PKIEngine      EngineType = "pki"
	DatabaseEngine EngineType = "database"
	SecretV1Engine EngineType = "kv"
	SecretV2Engine EngineType = "kv2"
	AWSEngine      EngineType = "aws"
	TransitEngine  EngineType = "transit"
)

func getLogger(level string) *zap.Logger {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logLevel, _ := zapcore.ParseLevel(level)
	config.Level = zap.NewAtomicLevelAt(logLevel)
	logger, _ := config.Build()
	return logger
}

func NewSecretEngine(v *vault.Client, e *SecretEngine, options *Options, et EngineType) Engine {

	// backup mode
	if options.BackupPath != "" {
		options.BackupPath = path.Join(options.BackupPath, e.Path+"."+string(et))

		if err := os.MkdirAll(options.BackupPath, 0755); err != nil {
			log.Fatalln(err)
		}
	}

	// restore mode
	if options.RestorePath != "" {
		ret := strings.TrimPrefix(path.Ext(path.Base(options.RestorePath)), ".")
		if ret != string(et) {
			log.Fatalln("Restore path does not match engine type")
		}
	}

	raw := &Raw{
		Vault:   v,
		Engine:  e,
		Options: options,
		L:       getLogger(options.LogLevel),
	}

	defer raw.L.Sync()

	switch et {
	case RawEngine:
		return nil
	case SSHEngine:
		return &SSH{raw}
	case TOTPEngine:
		return &TOTP{raw}
	case ADEngine:
		return &AD{raw}
	case PKIEngine:
		return &PKI{raw}
	case DatabaseEngine:
		return &Database{raw}
	case SecretV1Engine:
		return &SecretV1{raw}
	case SecretV2Engine:
		return &SecretV2{raw}
	case AWSEngine:
		return &AWS{raw}
	case TransitEngine:
		return &Transit{raw}
	}
	return nil
}
