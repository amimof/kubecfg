package main

import (
	"fmt"
	"os"

	"filippo.io/age"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/amimof/kubecfg/pkg/decrypt"
)

func newCompilerWithOptionalDecryptor(cfg *config.Config, identityFile string) (*config.Compiler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	compilerOpts := make([]config.CompilerOption, 0, 1)

	if cfg.HasEncryptedAuthInfos() {
		decryptor, err := loadAgeDecryptor(identityFile)
		if err != nil {
			return nil, err
		}
		compilerOpts = append(compilerOpts, config.WithDecryptor(decryptor))
	}

	return config.NewCompiler(baseDir, compilerOpts...), nil
}

func loadAgeDecryptor(identityFile string) (*decrypt.AgeDecryptor, error) {
	if identityFile != "" {
		f, err := os.Open(identityFile)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		identities, err := age.ParseIdentities(f)
		if err != nil {
			return nil, err
		}

		return decrypt.NewAgeDecryptor(identities...)
	}

	password, err := readPassword(os.Stdin)
	if err != nil {
		return nil, err
	}

	identity, err := age.NewScryptIdentity(password)
	if err != nil {
		return nil, err
	}

	return decrypt.NewAgeDecryptor(identity)
}
