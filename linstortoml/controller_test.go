package linstortoml_test

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/LINBIT/golinstor/linstortoml"
)

func TestController_Write(t *testing.T) {
	cases := []struct {
		config       linstortoml.Controller
		expectedToml string
	}{
		{
			config:       linstortoml.Controller{},
			expectedToml: "",
		},
		{
			config: linstortoml.Controller{
				Db:   &linstortoml.ControllerDb{ConnectionUrl: "https://127.0.0.1:2379", CaCertificate: "/path/to/bundle.pem"},
				Http: &linstortoml.ControllerHttp{Port: 5},
			},
			expectedToml: "[http]\nport = 5\n\n[db]\nconnection_url = \"https://127.0.0.1:2379\"\nca_certificate = \"/path/to/bundle.pem\"\n",
		},
	}

	for _, test := range cases {
		builder := strings.Builder{}
		enc := toml.NewEncoder(&builder)
		enc.Indent = ""
		err := enc.Encode(test.config)
		if err != nil {
			t.Fatalf("Could not write config: %v", err)
		}

		result := builder.String()
		if result != test.expectedToml {
			t.Fatalf("Mismatched config, expected '%+v', got '%+v'", test.expectedToml, result)
		}
	}
}
