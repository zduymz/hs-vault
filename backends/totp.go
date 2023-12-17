package backends

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"path"
)

type TOTP struct {
	*Object
}

type TOTPKey struct {
	Exported    bool   `json:"exported"`
	Url         string `json:"url"`
	Key         string `json:"key"`
	Issuer      string `json:"issuer"`
	AccountName string `json:"account_name"`
	Period      int    `json:"period"`
	Algorithm   any    `json:"algorithm"`
	Digits      int    `json:"digits"`
	Skew        int    `json:"skew"`
	QRSize      int    `json:"qr_size"`
}

func (t *TOTP) Backup(ctx context.Context) error {
	l := t.L.With(zap.String("method", "Backup"))
	if t.Options.RawAccessible {
		l.Debug("Start backup TOTP")
		keyPrefix := path.Join("logical", t.Engine.UUID)
		return t.RawBackup(ctx, keyPrefix, "key")
	}

	t.L.Warn("TOTP backup is not supported in normal mode")
	return nil
}

func (t *TOTP) Restore(ctx context.Context) error {
	l := t.L.With(zap.String("method", "Backup"))

	l.Debug("Start restore TOTP")
	paths, err := t.LocalWalk(ctx, t.Options.RestorePath, "key/")
	if err != nil {
		return err
	}

	for _, p := range paths {
		l.Debug("Restore TOTP key", zap.String("path", p))
		bs, err := t.ReadFileAndB64Decode(ctx, p)
		if err != nil {
			return err
		}

		var totpkey TOTPKey
		if err := json.Unmarshal(bs, &totpkey); err != nil {
			return err
		}
		// convert numeric algorithm to string
		switch totpkey.Algorithm.(float64) {
		case 0:
			totpkey.Algorithm = "SHA1"
		case 1:
			totpkey.Algorithm = "SHA256"
		case 2:
			totpkey.Algorithm = "SHA512"
		}

		tmp, err := json.Marshal(totpkey)
		if err != nil {
			return err
		}

		payload := map[string]interface{}{}
		if err := json.Unmarshal(tmp, &payload); err != nil {
			return err
		}

		if _, err := t.Vault.Write(ctx, path.Join(t.Engine.Path, "keys", path.Base(p)), payload); err != nil {
			return err
		}

	}
	return nil
}
