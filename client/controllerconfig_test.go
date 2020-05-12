package client

import (
	"github.com/BurntSushi/toml"
	"strings"
	"testing"
)

func TestControllerConfig_Write(t *testing.T) {
	cases := []struct {
		config       ControllerConfig
		expectedToml string
	}{{
		config:       ControllerConfig{},
		expectedToml: "[config]\n\n[debug]\n\n[log]\n\n[db]\n[db.etcd]\n\n[http]\n\n[https]\n\n[ldap]\n",
	}, {
		config: ControllerConfig{Db: ControllerConfigDb{ConnectionUrl: "https://127.0.0.1:2379", CaCertificate: "/path/to/bundle.pem"}, Http: ControllerConfigHttp{Port: 5}},
		expectedToml: "[config]\n\n[debug]\n\n[log]\n\n[db]\nconnection_url = \"https://127.0.0.1:2379\"\nca_certificate = \"/path/to/bundle.pem\"\n[db.etcd]\n\n[http]\nport = 5\n\n[https]\n\n[ldap]\n",
	}}

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
