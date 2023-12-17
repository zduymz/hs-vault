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
	RawAccessible  bool
}

type Mode string

type Engine interface {
	Backup(context.Context) error
	Restore(context.Context) error
}

type EngineType string

const (
	ADEngine       EngineType = "ad"
	AWSEngine      EngineType = "aws"
	DatabaseEngine EngineType = "database"
	PKIEngine      EngineType = "pki"
	RawEngine      EngineType = "raw"
	SSHEngine      EngineType = "ssh"
	SecretV1Engine EngineType = "kv"
	SecretV2Engine EngineType = "kv2"
	TOTPEngine     EngineType = "totp"
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

		if strings.HasSuffix(ret, "-r") {
			ret = strings.TrimSuffix(ret, "-r")
		}

		if ret != string(et) {
			log.Fatalln("Restore path does not match engine type")
		}
	}

	logger := getLogger(options.LogLevel)

	o := &Object{
		Vault:   v,
		Engine:  e,
		Options: options,
		L:       logger.With(zap.String("engine-path", e.Path), zap.String("engine-type", string(et))),
	}

	defer o.L.Sync()

	if et == SecretV2Engine || et == TransitEngine {
		options.RawAccessible = false
	}

	switch et {
	case RawEngine:
		return nil
	case SSHEngine:
		return &SSH{o}
	case TOTPEngine:
		return &TOTP{o}
	case ADEngine:
		return &AD{o}
	case PKIEngine:
		return &PKI{o}
	case DatabaseEngine:
		return &Database{o}
	case SecretV1Engine:
		return &SecretV1{o}
	case SecretV2Engine:
		return &SecretV2{o}
	case AWSEngine:
		return &AWS{o}
	case TransitEngine:
		return &Transit{o}
	}
	return nil
}
