package decrypt

import (
	"bytes"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

type AgeEncryptor struct {
	Recipients []age.Recipient
}

func NewAgeEncryptor(recipients ...age.Recipient) (*AgeEncryptor, error) {
	return &AgeEncryptor{Recipients: recipients}, nil
}

func (e *AgeEncryptor) EncryptString(plaintext string) (string, error) {
	var buf bytes.Buffer

	armorWriter := armor.NewWriter(&buf)
	w, err := age.Encrypt(armorWriter, e.Recipients...)
	if err != nil {
		return "", err
	}

	if _, err := io.WriteString(w, plaintext); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	if err := armorWriter.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
