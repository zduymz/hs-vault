package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault-client-go"
	"github.com/mitchellh/mapstructure"
	"github.com/urfave/cli/v2"
	"github.com/zduymz/hs-vault/backends"
	"log"
	"os"
	"strings"
)

type SecretEngineResponse struct {
	Accessor    string
	Description string
	Local       bool
	Type        string
	Uuid        string
	Options     map[string]interface{}
}

func (engine *SecretEngineResponse) getEngineType() backends.EngineType {
	if engine.Type == "kv" {
		if version, ok := engine.Options["version"]; ok && version.(string) == "2" {
			return backends.SecretV2Engine
		}
		//if engine.Options["version"].(string) == "2" {
		//	return backends.SecretV2Engine
		//}
		return backends.SecretV1Engine
	}
	return backends.EngineType(engine.Type)
}

func listEngines(v *vault.Client) (map[string]SecretEngineResponse, error) {
	var secretEngines = make(map[string]SecretEngineResponse)
	var ctx = context.Background()
	engines, err := v.System.MountsListSecretsEngines(ctx)
	if err != nil {
		return nil, err
	}

	for key, value := range engines.Data {
		key = strings.TrimSuffix(key, "/")
		output := SecretEngineResponse{}
		err := mapstructure.Decode(value, &output)
		if err != nil {
			log.Fatalln(err)
		}

		if output.Type == "system" || output.Type == "cubbyhole" || output.Type == "identity" {
			continue
		}

		if output.Type == "consul" {
			continue
		}

		if output.Type == "generic" {
			continue
		}
		secretEngines[key] = output
	}
	return secretEngines, nil
}

func getVaultClient() *vault.Client {
	client, err := vault.New(vault.WithEnvironment())
	if err != nil {
		log.Fatalln(err)
	}
	if err := client.SetToken(os.Getenv("VAULT_TOKEN")); err != nil {
		log.Fatalln(err)
	}
	return client
}

func backup(c *cli.Context) error {
	client := getVaultClient()
	engines, err := listEngines(client)
	if err != nil {
		log.Fatalln(err)
	}

	// namespace is set
	if c.IsSet(FlagNamespace) {
		if err := client.SetNamespace(c.String(FlagNamespace)); err != nil {
			log.Fatalln(err)
		}
	}

	// backup specific path
	if c.IsSet(FlagPath) {
		key := c.String(FlagPath)
		engine, ok := engines[key]
		if !ok {
			log.Fatalf("Engine with path '%v' not found", key)
		}

		se := backends.NewSecretEngine(client,
			&backends.SecretEngine{
				Path: key,
				Type: engine.Type,
				UUID: engine.Uuid,
			},
			&backends.Options{
				Base64Encode: c.Bool(FlagB64Encode),
				BackupPath:   c.String(FlagDest),
				LogLevel:     c.String(FlagLogLevel),
			},
			engine.getEngineType(),
		)

		err := se.Backup(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		return nil
	}

	//default backup all engines
	for key, engine := range engines {
		ss := backends.NewSecretEngine(client,
			&backends.SecretEngine{
				Path: key,
				Type: engine.Type,
				UUID: engine.Uuid,
			},
			&backends.Options{
				Base64Encode: c.Bool(FlagB64Encode),
				BackupPath:   c.String(FlagDest),
				LogLevel:     c.String(FlagLogLevel),
			},
			engine.getEngineType(),
		)
		err := ss.Backup(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
	}

	return nil
}

func restore(c *cli.Context) error {
	client := getVaultClient()
	engines, err := listEngines(client)
	if err != nil {
		log.Fatalln(err)
	}

	// namespace is set
	if c.IsSet(FlagNamespace) {
		if err := client.SetNamespace(c.String(FlagNamespace)); err != nil {
			log.Fatalln(err)
		}
	}

	// backup specific path
	if c.IsSet(FlagPath) {
		key := c.String(FlagPath)
		engine, ok := engines[key]
		if !ok {
			log.Fatalf("Engine with path '%v' not found", key)
		}

		se := backends.NewSecretEngine(client,
			&backends.SecretEngine{
				Path: key,
				Type: engine.Type,
				UUID: engine.Uuid,
			},
			&backends.Options{
				Base64Encode: c.Bool(FlagB64Encode),
				LogLevel:     c.String(FlagLogLevel),
				RestorePath:  c.String(FlagSource),
			},
			engine.getEngineType(),
		)

		err := se.Restore(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		return nil
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:        "hs-vault",
		Usage:       "Another tool to backup and restore Hashicorp Vault secrets engines",
		Description: description,
		Commands:    getCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error running command: %v\n", err)
		os.Exit(1)
	}
}

const description = `
This is another tool to backup and restore Hashicorp Vault secrets engines.
Not officially supported by Hashicorp.

Usage:

Backup all engines:
	$ hs-vault backup

Backup single engine:
	$ hs-vault backup -p <engine_path>

Backup single engine to specific directory with vault namespace
	$ hs-vault backup -p <engine_path> -d <backup_dir> -n <vault_namespace>

Restore all engines:
	$ hs-vault restore -s <backup_dir>

Restore single engine:
	$ hs-vault restore -p <engine_path> -s <backup_dir>/<engine_path>.<engine_type>
`
