package linstortoml

import lapi "github.com/LINBIT/golinstor/client"

type Controller struct {
	Http    *ControllerHttp    `toml:"http,omitempty"`
	Https   *ControllerHttps   `toml:"https,omitempty"`
	Ldap    *ControllerLdap    `toml:"ldap,omitempty"`
	Db      *ControllerDb      `toml:"db,omitempty"`
	Logging *ControllerLogging `toml:"logging,omitempty"`
	Encrypt *ControllerEncrypt `toml:"encrypt,omitempty"`
	WebUi   *ControllerWebUi   `toml:"webUi,omitempty"`
}

type ControllerHttp struct {
	Enabled    *bool  `toml:"enabled,omitempty"`
	ListenAddr string `toml:"listen_addr,omitempty"`
	Port       int    `toml:"port,omitzero"`
}

type ControllerHttps struct {
	Enabled            *bool  `toml:"enabled,omitempty"`
	ListenAddr         string `toml:"listen_addr,omitempty"`
	Port               int    `toml:"port,omitzero"`
	Keystore           string `toml:"keystore,omitempty"`
	KeystorePassword   string `toml:"keystore_password,omitempty"`
	Truststore         string `toml:"truststore,omitempty"`
	TruststorePassword string `toml:"truststore_password,omitempty"`
}

type ControllerLdap struct {
	Enabled           *bool  `toml:"enabled,omitempty"`
	AllowPublicAccess *bool  `toml:"allow_pubic_access,omitempty"`
	Uri               string `toml:"uri,omitempty"`
	Dn                string `toml:"dn,omitempty"`
	SearchBase        string `toml:"search_base,omitempty"`
	SearchFilter      string `toml:"search_filter,omitempty"`
}

type ControllerDb struct {
	User              string            `toml:"user,omitempty"`
	Password          string            `toml:"password,omitempty"`
	ConnectionUrl     string            `toml:"connection_url,omitempty"`
	CaCertificate     string            `toml:"ca_certificate,omitempty"`
	ClientCertificate string            `toml:"client_certificate,omitempty"`
	ClientKeyPkcs8Pem string            `toml:"client_key_pkcs8_pem,omitempty"`
	ClientKeyPassword string            `toml:"client_key_password,omitempty"`
	Etcd              *ControllerDbEtcd `toml:"etcd,omitempty"`
}

type ControllerDbEtcd struct {
	OpsPerTransaction int    `toml:"ops_per_transaction,omitzero"`
	Prefix            string `toml:"prefix,omitempty"`
}

type ControllerLogging struct {
	Level             lapi.LogLevel `toml:"level,omitempty"`
	LinstorLevel      lapi.LogLevel `toml:"linstor_level,omitempty"`
	RestAccessLogPath string        `toml:"rest_access_log_path,omitempty"`
	RestAccessLogMode string        `toml:"rest_access_log_mode,omitempty"`
}

type ControllerEncrypt struct {
	Passphrase string `toml:"passphrase,omitempty"`
}

type ControllerWebUi struct {
	Directory string `toml:"directory,omitempty"`
}
