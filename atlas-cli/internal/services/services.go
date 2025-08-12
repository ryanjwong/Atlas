package services

import (
	"fmt"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
)

type Services struct {
	verbose       bool
	output        string
	version       string
	localProvider *providers.LocalProvider
}

func NewServices(verbose bool, output string, version string) *Services {
	localProvider := providers.NewLocalProvider()

	return &Services{
		verbose:       verbose,
		output:        output,
		version:       version,
		localProvider: localProvider,
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
