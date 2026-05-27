package main

import (
	"errors"
	"io"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptCmdRequiresArgument(t *testing.T) {
	cmd := newEncryptCmd()
	err := cmd.Args(cmd, nil)
	require.Error(t, err)
}

func TestRunEncryptCmdReturnsScryptRecipientError(t *testing.T) {
	originalReadPassword := readPasswordFn
	originalNewScryptRecipient := newScryptRecipientFn
	t.Cleanup(func() {
		readPasswordFn = originalReadPassword
		newScryptRecipientFn = originalNewScryptRecipient
	})

	readPasswordFn = func(r io.Reader) (string, error) {
		return "password", nil
	}
	expectedErr := errors.New("recipient failure")
	newScryptRecipientFn = func(password string) (age.Recipient, error) {
		return nil, expectedErr
	}

	err := runEncryptCmd("plaintext", "")
	require.ErrorIs(t, err, expectedErr)
}
