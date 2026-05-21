package config

import (
	"fmt"
	"time"
)

func (d *ConnectionConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type rawConnectionConfig struct {
		Transport   string `yaml:"transport"`
		Address     string `yaml:"address"`
		UnitID      byte   `yaml:"unit_id"`
		Timeout     string `yaml:"timeout"`
		IdleTimeout string `yaml:"idle_timeout"`
	}
	var raw rawConnectionConfig
	if err := unmarshal(&raw); err != nil {
		return err
	}
	d.Transport = raw.Transport
	d.Address = raw.Address
	d.UnitID = raw.UnitID
	var err error
	if raw.Timeout != "" {
		d.Timeout, err = time.ParseDuration(raw.Timeout)
		if err != nil {
			return fmt.Errorf("parse timeout %q: %w", raw.Timeout, err)
		}
	}
	if raw.IdleTimeout != "" {
		d.IdleTimeout, err = time.ParseDuration(raw.IdleTimeout)
		if err != nil {
			return fmt.Errorf("parse idle_timeout %q: %w", raw.IdleTimeout, err)
		}
	}
	return nil
}

func (d ConnectionConfig) MarshalYAML() (any, error) {
	return struct {
		Transport   string `yaml:"transport"`
		Address     string `yaml:"address"`
		UnitID      byte   `yaml:"unit_id"`
		Timeout     string `yaml:"timeout"`
		IdleTimeout string `yaml:"idle_timeout"`
	}{
		Transport:   d.Transport,
		Address:     d.Address,
		UnitID:      d.UnitID,
		Timeout:     d.Timeout.String(),
		IdleTimeout: d.IdleTimeout.String(),
	}, nil
}
