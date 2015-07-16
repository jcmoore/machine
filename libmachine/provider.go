package libmachine

import (
	"fmt"
	"os"
	"errors"

	"github.com/docker/machine/drivers"
)

type Provider struct {
	store Store
	driverConfig    drivers.DriverOptions
	driverSpecifier drivers.OptionsSpecifier
}

func New(store Store, config drivers.DriverOptions, specifier drivers.OptionsSpecifier) (*Provider, error) {
	return &Provider{
		store: store,
		driverConfig: config,
		driverSpecifier: specifier,
	}, nil
}

func (provider *Provider) getDriverConfig(driver string) (drivers.DriverOptions, error) {
	if provider.driverSpecifier != nil {
		driverConfig, err := provider.driverSpecifier.SpecifyFlags(driver, provider.driverConfig)
		return driverConfig, err
	} else {
		return provider.driverConfig, nil;
	}
}

func (provider *Provider) Create(name string, driverName string, hostOptions *HostOptions) (*Host, error) {
	validName := ValidateHostName(name)
	if !validName {
		return nil, ErrInvalidHostname
	}
	exists, err := provider.store.Exists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("Machine %s already exists", name)
	}

	host, err := NewHost(name, driverName, hostOptions)
	if err != nil {
		return host, err
	}

	driverConfig, err := provider.getDriverConfig(driverName)
	if err != nil {
		return host, err
	}

	if driverConfig != nil {
		if err := host.SetDriverConfigFromFlags(driverConfig); err != nil {
			return host, err
		}
	}

	if err := host.Prepare(); err != nil {
		return nil, err
	}

	if err := host.SaveConfig(); err != nil {
		return host, err
	}

	if err := host.Create(name); err != nil {
		return host, err
	}

	return host, nil
}

func (provider *Provider) Exists(name string) (bool, error) {
	return provider.store.Exists(name)
}

func (provider *Provider) GetActive() (*Host, error) {
	hosts, err := provider.List()
	if err != nil {
		return nil, err
	}

	dockerHost := os.Getenv("DOCKER_HOST")
	hostListItems := GetHostListItems(hosts)

	for _, item := range hostListItems {
		if dockerHost == item.URL {
			host, err := provider.store.Get(item.Name)
			if err != nil {
				return nil, err
			}
			return host, nil
		}
	}

	return nil, errors.New("Active host not found")
}

func (provider *Provider) List() ([]*Host, error) {
	hosts, fatal := provider.store.List()

	if fatal != nil {
		return nil, fatal
	}

	for _, host := range hosts {
		driverConfig, err := provider.getDriverConfig(host.DriverName)
	        if err != nil {
	                fatal = err
	        }

		if driverConfig != nil {
		        if err := host.SetDriverConfigFromFlags(driverConfig); err != nil {
				if fatal != nil {
					fatal = err
				}
		        }
		}
	}

	return hosts, fatal
}

func (provider *Provider) Get(name string) (*Host, error) {
	return provider.store.Get(name)
}

func (provider *Provider) Remove(name string, force bool) error {
	host, err := provider.store.Get(name)

	if err != nil {
		return err
	}

	driverConfig, err := provider.getDriverConfig(host.DriverName)
        if err != nil {
                return err
        }

	if driverConfig != nil {
	        if err := host.SetDriverConfigFromFlags(driverConfig); err != nil {
	                return err
	        }
	}

	if err := host.Remove(force); err != nil {
		if !force {
			return err
		}
	}

	return provider.store.Remove(name, force)
}
