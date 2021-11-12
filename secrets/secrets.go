package secrets

import (
	"errors"

	"github.com/hashicorp/go-hclog"
)

// Define constant key names for SecretsManagerParams.Params
const (
	// Path is the path to the base working directory
	Path = "path"

	// Token is the token used for authenticating with a KMS
	Token = "token"

	// Server is the address of the KMS
	Server = "server"

	// Name is the name of the current node
	Name = "name"
)

// Define constant names for available secrets
const (
	// ValidatorKey is the private key secret of the validator node
	ValidatorKey = "validator-key"

	// NetworkKey is the libp2p private key secret used for networking
	NetworkKey = "network-key"
)

// Define constant file names for the local StorageManager
const (
	ValidatorKeyLocal = "validator.key"
	NetworkKeyLocal   = "libp2p.key"
)

// Define constant folder names for the local StorageManager
const (
	ConsensusFolderLocal = "consensus"
	NetworkFolderLocal   = "libp2p"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
)

type SecretsManagerType string

// Define constant types of secrets managers
const (
	// Local pertains to the local FS [Default]
	Local SecretsManagerType = "local"

	// HashicorpVault pertains to the Hashicorp Vault server
	HashicorpVault SecretsManagerType = "hashicorp-vault"
)

// SecretsManager defines the base public interface that all
// secret manager implementations should have
type SecretsManager interface {
	// Setup performs secret manager-specific setup
	Setup() error

	// GetSecret gets the secret by name
	GetSecret(name string) (interface{}, error)

	// SetSecret sets the secret to a provided value
	SetSecret(name string, value interface{}) error

	// HasSecret checks if the secret is present
	HasSecret(name string) bool

	// RemoveSecret removes the secret from storage
	RemoveSecret(name string) error
}

// SecretsManagerParams defines the configuration params for the
// secrets manager
type SecretsManagerParams struct {
	// Local logger object
	Logger hclog.Logger

	// Params needed for the SecretsManager to function
	Params map[string]interface{}
}

// SecretsManagerFactory is the factory method for secrets managers
type SecretsManagerFactory func(
	config *SecretsManagerParams,
) (SecretsManager, error)

// SupportedServiceManager checks if the passed in service manager type is supported
func SupportedServiceManager(service SecretsManagerType) bool {
	return service == HashicorpVault ||
		service == Local
}
