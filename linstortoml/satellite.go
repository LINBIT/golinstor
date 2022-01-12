package linstortoml

import lapi "github.com/LINBIT/golinstor/client"

type Satellite struct {
	NetCom  *SatelliteNetCom  `toml:"netcom,omitempty"`
	Logging *SatelliteLogging `toml:"logging,omitempty"`
	Files   *SatelliteFiles   `toml:"files,omitempty"`
}

type SatelliteNetCom struct {
	Type                string `toml:"type,omitempty"`
	BindAddress         string `toml:"bind_address,omitempty"`
	Port                int    `toml:"port,omitzero"`
	ServerCertificate   string `toml:"server_certificate,omitempty"`
	KeyPassword         string `toml:"key_password,omitempty"`
	KeystorePassword    string `toml:"keystore_password,omitempty"`
	TrustedCertificates string `toml:"trusted_certificates,omitempty"`
	TruststorePassword  string `toml:"truststore_password,omitempty"`
	SslProtocol         string `toml:"ssl_protocol,omitempty"`
}

type SatelliteLogging struct {
	Level        lapi.LogLevel `toml:"level,omitempty"`
	LinstorLevel lapi.LogLevel `toml:"linstor_level,omitempty"`
}

type SatelliteFiles struct {
	AllowExtFiles []string `toml:"allowExtFiles,omitempty"`
}
