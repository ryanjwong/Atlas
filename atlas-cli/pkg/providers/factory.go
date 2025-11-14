package providers

import (
	"fmt"
)

type ProviderFactory struct {
	providers map[string]func(region, profile string) Provider
}

func NewProviderFactory() *ProviderFactory {
	factory := &ProviderFactory{
		providers: make(map[string]func(region, profile string) Provider),
	}
	
	factory.RegisterProvider("local", func(region, profile string) Provider {
		return NewLocalProvider()
	})
	
	factory.RegisterProvider("aws", func(region, profile string) Provider {
		return NewAWSProvider(profile, region)
	})
	
	return factory
}

func (f *ProviderFactory) RegisterProvider(name string, constructor func(region, profile string) Provider) {
	f.providers[name] = constructor
}

func (f *ProviderFactory) CreateProvider(name, region, profile string) (Provider, error) {
	constructor, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}
	
	if region == "" {
		switch name {
		case "local":
			region = "local"
		case "aws":
			region = "us-west-2"
		default:
			region = "default"
		}
	}
	
	return constructor(region, profile), nil
}

func (f *ProviderFactory) GetSupportedProviders() []string {
	var providers []string
	for name := range f.providers {
		providers = append(providers, name)
	}
	return providers
}

func GetDefaultProviderFactory() *ProviderFactory {
	return NewProviderFactory()
}