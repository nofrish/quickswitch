package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed codex_default_config.toml
var codexDefaultConfigTOML []byte

func scaffoldCodexDefaultConfig(dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	return os.WriteFile(dst, codexDefaultConfigTOML, 0600)
}
