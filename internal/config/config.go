package config

import (
	"fmt"
	"os"
	"path"

	"github.com/dalibo/ldap2pg/internal/inspect"
	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/roles"
	"github.com/lithammer/dedent"
	"golang.org/x/exp/slog"
)

func FindConfigFile(userValue string) (configpath string) {
	if "" != userValue {
		return userValue
	}

	slog.Debug("Searching configuration file in standard locations.")
	home, _ := os.UserHomeDir()
	candidates := []string{
		"./ldap2pg.yml",
		"./ldap2pg.yaml",
		path.Join(home, "/.config/ldap2pg.yml"),
		path.Join(home, "/.config/ldap2pg.yaml"),
		"/etc/ldap2pg.yml",
		"/etc/ldap2pg.yaml",
	}

	for _, candidate := range candidates {
		_, err := os.Stat(candidate)
		if err == nil {
			slog.Debug("Found configuration file.",
				"path", candidate)

			return candidate
		}
		slog.Debug("Ignoring configuration file.",
			"path", candidate,
			"error", err)
	}

	return ""
}

// Config holds the YAML configuration. Not the flags.
type Config struct {
	Version  int
	Ldap     LdapConfig
	Postgres inspect.Config
	SyncMap  SyncMap `mapstructure:"sync_map"`
}

type LdapConfig struct {
	URI      string
	BindDn   string
	Password string
}

type LdapSearch struct {
	Base        string
	Scope       ldap.Scope
	Filter      string
	Attributes  []string
	Subsearches map[string]Subsearch `mapstructure:"joins"`
}

type Subsearch struct {
	Filter     string
	Scope      ldap.Scope
	Attributes []string
}

type RoleRule struct {
	Name    pyfmt.Format
	Options roles.Options
	Comment pyfmt.Format
	Parents []pyfmt.Format
}

func (r RoleRule) IsStatic() bool {
	if 0 < len(r.Name.Fields) {
		return false
	}
	if 0 < len(r.Comment.Fields) {
		return false
	}
	for _, f := range r.Parents {
		if 0 < len(f.Fields) {
			return false
		}
	}
	return true
}

// New initiate a config structure with defaults.
func New() Config {
	return Config{
		Postgres: inspect.Config{
			DatabasesQuery: inspect.RowsOrSQL{Value: dedent.Dedent(`
			SELECT datname FROM pg_catalog.pg_database
			WHERE datallowconn IS TRUE ORDER BY 1;`)},
			RolesBlacklistQuery: inspect.RowsOrSQL{Value: []interface{}{"pg_*", "postgres"}},
		},
	}
}

func Load(path string) (Config, error) {
	c := New()
	err := c.Load(path)
	return c, err
}

func (c *Config) Load(path string) (err error) {
	slog.Debug("Loading YAML configuration.")

	yamlData, err := ReadYaml(path)
	if err != nil {
		return
	}
	err = c.checkVersion(yamlData)
	if err != nil {
		return
	}
	root, err := NormalizeConfigRoot(yamlData)
	if err != nil {
		return fmt.Errorf("YAML error: %w", err)
	}
	err = c.LoadYaml(root)
	if err != nil {
		return
	}

	c.SyncMap = c.SyncMap.SplitStaticRules()
	return
}
