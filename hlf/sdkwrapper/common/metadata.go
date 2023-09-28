package common

// VaultConfig is a struct with vault config
type VaultConfig struct {
	// Token is a token for vault access
	Token string `mapstructure:"token,omitempty"`
	// Address is an address of vault
	Path string `mapstructure:"path,omitempty"`
	// CAcert is a path to CA cert for vault
	CAcert string `mapstructure:"ca_cert,omitempty"`
	// Address is an address of vault (e.g. https://vault.example.com:8200)
	Address string `mapstructure:"address,omitempty"`
	// UserCertName is a name of user cert for vault (e.g. User1@org1.example.com-cert.pem)
	UserCertName string `mapstructure:"user_cert_name,omitempty"`
}
