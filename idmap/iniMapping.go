package idmap

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type IniMapping struct {
	cfg      *ini.File
	filePath string
}

// NewIniMapping creates a new IniMapping instance with default or provided filePath.
func NewIniMapping(optionalFilePath ...string) (*IniMapping, error) {
	defaultPath := filepath.Join(".", "idmap.ini")
	if len(optionalFilePath) > 0 {
		defaultPath = optionalFilePath[0]
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowBooleanKeys: true,
	}, defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = ini.Empty()
		} else {
			return nil, err
		}
	}

	return &IniMapping{
		cfg:      cfg,
		filePath: defaultPath,
	}, nil
}

// WriteConfig writes a value into the specified section and key.
func (m *IniMapping) WriteConfig(sectionName, keyName, value string) error {
	section, err := m.cfg.NewSection(sectionName)
	if err != nil {
		return err
	}

	_, err = section.NewKey(keyName, value)
	if err != nil {
		return err
	}

	return m.cfg.SaveToIndent(m.filePath, "\t")
}

// ReadConfig reads a value from the specified section and key.
func (m *IniMapping) ReadConfig(sectionName, keyName string) (string, error) {
	section := m.cfg.Section(sectionName)
	if section == nil {
		return "", fmt.Errorf("section '%s' does not exist", sectionName)
	}

	key := section.Key(keyName)
	if key == nil {
		return "", fmt.Errorf("key '%s' in section '%s' does not exist", keyName, sectionName)
	}

	return key.String(), nil
}
