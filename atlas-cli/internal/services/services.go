package services

import (
	"fmt"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
	"github.com/ryanjwong/Atlas/atlas-cli/pkg/state"
)

type Services struct {
	verbose       bool
	output        string
	version       string
	stateManager  state.StateManager
	localProvider providers.LocalProvider
}

func NewServices(verbose bool, output string, version string, path string) (*Services, error) {
	stateManager, err := state.NewSQLiteStateManager(path)
	if err != nil {
		return nil, fmt.Errorf("error initializing state manager with path %s: %s", path, err)
	}
	return &Services{
		verbose:       verbose,
		output:        output,
		version:       version,
		stateManager:  stateManager,
		localProvider: providers.LocalProvider{},
	}, nil
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

func (s *Services) GetStateManager() state.StateManager {
	return s.stateManager
}

func (s *Services) GetLocalProvider() providers.LocalProvider {
	return s.localProvider
}
