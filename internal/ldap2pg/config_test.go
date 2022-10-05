package ldap2pg_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/ldap2pg"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (suite *ConfigSuite) TestLoadEnvDoesNotOverwriteConfigFile() {
	r := suite.Require()

	config := ldap2pg.Config{
		ConfigFile: "defined-ldap2pg.yaml",
	}
	values := ldap2pg.EnvValues{
		ConfigFile: "env-ldap2pg.yaml",
	}
	config.LoadEnv(values)

	r.Equal(config.ConfigFile, "defined-ldap2pg.yaml")
}

func TestConfig(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	suite.Run(t, new(ConfigSuite))
}