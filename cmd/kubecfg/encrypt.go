package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
	"github.com/amimof/kubecfg/pkg/decrypt"
	"github.com/spf13/cobra"
)

var (
	readPasswordFn       = readPassword
	newScryptRecipientFn = func(password string) (age.Recipient, error) {
		return age.NewScryptRecipient(password)
	}
)

func newEncryptCmd() *cobra.Command {
	var publicKey string
	cmd := &cobra.Command{
		Use:   "encrypt [STRING]",
		Short: "Encrypt a value for kubecfg.yaml",
		Long:  `Encrypt a string and print the result for use in kubecfg.yaml.`,
		Example: `  kubecfg encrypt "secret-value"
  kubecfg encrypt "secret-value" --public-key age1...`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runEncryptCmd(args[0], publicKey)
		}),
	}

	cmd.Flags().StringVar(&publicKey, "public-key", "", "encrypt using public key")

	return cmd
}

func runEncryptCmd(input string, publicKey string) error {
	var recipient age.Recipient

	if publicKey != "" {
		rec, err := age.ParseX25519Recipient(publicKey)
		if err != nil {
			return err
		}
		recipient = rec
	} else {
		password, err := readPasswordFn(os.Stdin)
		if err != nil {
			return err
		}
		recipient, err = newScryptRecipientFn(password)
		if err != nil {
			return err
		}
	}

	encryptor, err := decrypt.NewAgeEncryptor(recipient)
	if err != nil {
		return err
	}

	encrypted, err := encryptor.EncryptString(input)
	if err != nil {
		return err
	}

	fmt.Println(encrypted)
	return nil
}

func readPassword(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	fmt.Print("Password: ")
	password, err := br.ReadString('\n')
	if err != nil {
		return "", errors.New("expected password on first line of stdin")
	}

	password = trimLineEnding(password)
	if password == "" {
		return "", errors.New("password must not be empty")
	}

	// payload, err := io.ReadAll(br)
	// if err != nil {
	// 	return "", nil, err
	// }
	//
	// if len(payload) == 0 {
	// 	return "", nil, errors.New("expected payload after password line")
	// }

	return password, nil
}

func trimLineEnding(s string) string {
	s = bytes.NewBufferString(s).String()

	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}

	return s
}
