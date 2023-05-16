package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// Marshall YAML from file path or stdin if path is -.
func ReadYaml(path string) (values interface{}, err error) {
	var fo io.ReadCloser
	if path == "-" {
		slog.Info("Reading configuration from standard input.")
		fo = os.Stdin
	} else {
		fo, err = os.Open(path)
		if err != nil {
			return
		}
	}
	defer fo.Close()
	dec := yaml.NewDecoder(fo)
	err = dec.Decode(&values)
	return
}

// Fill configuration from YAML data.
func (config *Config) LoadYaml(root map[string]interface{}) (err error) {
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		_ = encoder.Encode(root)
		encoder.Close()
		slog.Debug("Normalized YAML:\n" + buf.String())
	}

	err = DecodeYaml(root, config)
	if err != nil {
		return
	}

	for i := range config.SyncMap {
		item := &config.SyncMap[i]
		item.InferAttributes()
		// states.ComputeWanted is simplified base on the assumption
		// there is no more than one sub-search. Fail otherwise.
		if 1 < len(item.LdapSearch.Subsearches) {
			err = fmt.Errorf("multiple sub-search unsupported")
			return
		}
		item.ReplaceAttributeAsSubentryField()
	}

	slog.Debug("Loaded configuration file.", "version", config.Version)
	return
}

// Wrap mapstructure for config object
func DecodeYaml(yaml any, c *Config) (err error) {
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: decodeMapHook,
		Metadata:   &mapstructure.Metadata{},
		Result:     c,
	})
	if err == nil {
		err = d.Decode(yaml)
	}
	return
}

// Decode custom types for mapstructure. Implements mapstructure.DecodeHookFuncValue.
func decodeMapHook(from, to reflect.Value) (interface{}, error) {
	switch to.Type() {
	case reflect.TypeOf(pyfmt.Format{}):
		f := to.Interface().(pyfmt.Format)
		err := f.Parse(from.String())
		if err != nil {
			return nil, err
		}
		return f, nil
	case reflect.TypeOf(roles.Options{}):
		r := to.Interface().(roles.Options)
		r.LoadYaml(from.Interface().(map[string]interface{}))
		return r, nil
	case reflect.TypeOf(inspect.RowsOrSQL{}):
		switch from.Interface().(type) {
		case string:
			return inspect.RowsOrSQL{Value: from.String()}, nil
		case []interface{}:
			return inspect.RowsOrSQL{Value: from.Interface()}, nil
		default:
			return nil, fmt.Errorf("bad YAML for query")
		}
	case reflect.TypeOf(ldap.Scope(1)):
		s, err := ldap.ParseScope(from.String())
		if err != nil {
			return from.Interface(), err
		}
		return s, nil
	}
	return from.Interface(), nil
}

func (config *Config) checkVersion(yaml interface{}) (err error) {
	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		return errors.New("YAML is not a map")
	}
	version, ok := yamlMap["version"]
	if !ok {
		slog.Debug("Fallback to version 5.")
		version = 5
	}
	config.Version, ok = version.(int)
	if !ok {
		return errors.New("Configuration version must be integer")
	}
	if config.Version != 5 {
		slog.Debug("Unsupported configuration version.", "version", config.Version)
		return errors.New("Unsupported configuration version")
	}
	return
}
