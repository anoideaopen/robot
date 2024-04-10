package common

type VaultConfig struct {
	Token        string `mapstructure:"token,omitempty"`
	Path         string `mapstructure:"path,omitempty"`
	CAcert       string `mapstructure:"ca_cert,omitempty"`
	Address      string `mapstructure:"address,omitempty"`
	UserCertName string `mapstructure:"user_cert_name,omitempty"` // User1@org1.example.com-cert.pem
}
