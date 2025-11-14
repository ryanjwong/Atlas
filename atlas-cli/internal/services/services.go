package services

import (
	"fmt"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
)

type Services struct {
	verbose         bool
	output          string
	version         string
	providerFactory *providers.ProviderFactory
	localProvider   *providers.LocalProvider
}

func NewServices(verbose bool, output string, version string) *Services {
	localProvider := providers.NewLocalProvider()
	providerFactory := providers.GetDefaultProviderFactory()

	return &Services{
		verbose:         verbose,
		output:          output,
		version:         version,
		providerFactory: providerFactory,
		localProvider:   localProvider,
	}
}

func (s *Services) GetVerbose() bool {
	return s.verbose
}

func (s *Services) GetOutput() string {
	return s.output
}

func (s *Services) GetVersion() string {
	return s.version
}

func (s *Services) Log(message string) {
	if s.verbose {
		fmt.Printf("[DEBUG] %s\n", message)
	}
}


func (s *Services) GetLocalProvider() *providers.LocalProvider {
	return s.localProvider
}

func (s *Services) GetProvider(providerName, region, profile string) (providers.Provider, error) {
	return s.providerFactory.CreateProvider(providerName, region, profile)
}

func (s *Services) GetProviderFactory() *providers.ProviderFactory {
	return s.providerFactory
}

func (s *Services) GetSupportedProviders() []string {
	return s.providerFactory.GetSupportedProviders()
}
