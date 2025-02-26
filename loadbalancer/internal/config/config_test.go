package config

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Kostaaa1/loadbalancer/internal/fileutil"
)

func TestReadConfig(t *testing.T) {
	p, err := fileutil.ConfigPath()
	if err != nil {
		fmt.Println(err)
		return
	}

	cfg, err := Load(p)
	if err != nil {
		fmt.Println(err)
		return
	}

	b, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}
