package decrypt

import (
	"bytes"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

type AgeDecryptor struct {
	Identities []age.Identity
}

func NewAgeDecryptor(identities ...age.Identity) (*AgeDecryptor, error) {
	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities configured")
	}
	return &AgeDecryptor{Identities: identities}, nil
}

func (d *AgeDecryptor) DecryptString(ciphertext string) (string, error) {
	plaintext, err := d.DecryptBytes([]byte(ciphertext))
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (d *AgeDecryptor) DecryptBytes(ciphertext []byte) ([]byte, error) {
	r := bytes.NewReader(ciphertext)

	// Try armored first.
	armored := armor.NewReader(r)

	plaintextReader, err := age.Decrypt(armored, d.Identities...)
	if err != nil {
		// Fallback to binary age.
		_, err := r.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}

		plaintextReader, err = age.Decrypt(r, d.Identities...)
		if err != nil {
			return nil, err
		}
	}

	return io.ReadAll(plaintextReader)
}
