package fileutil

import (
	"os"
	"path/filepath"
)

func ConfigPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		js := filepath.Join(wd, "lb_config.json")
		if _, err := os.Stat(js); err == nil {
			return js, err
		}
		yml := filepath.Join(wd, "lb_config.yml")
		if _, err := os.Stat(yml); err == nil {
			return yml, err
		}
		xml := filepath.Join(wd, "lb_config.xml")
		if _, err := os.Stat(xml); err == nil {
			return xml, err
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			return "", os.ErrNotExist
		}
		wd = parent
	}
}
