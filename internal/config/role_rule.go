package config

import (
	"fmt"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slog"
)

type RoleRule struct {
	Names    []string
	Options  RoleOptions
	Comments []string
	Parents  mapset.Set[string]
}

func (rule *RoleRule) LoadYaml(yaml map[string]interface{}) {
	rule.Names = yaml["names"].([]string)
	rule.Comments = yaml["comments"].([]string)
	rule.Options.LoadYaml(yaml["options"].(map[string]interface{}))
	rule.Parents = mapset.NewSet[string](yaml["parents"].([]string)...)
}

type RoleOptions struct {
	Super       bool `column:"rolsuper" token:"SUPERUSER"`
	CreateDB    bool `column:"rolcreatedb" token:"CREATEDB"`
	CreateRole  bool `column:"rolcreaterole" token:"CREATEROLE"`
	Inherit     bool `column:"rolinherit" token:"INHERIT"`
	CanLogin    bool `column:"rolcanlogin" token:"LOGIN"`
	Replication bool `column:"rolreplication" token:"REPLICATION"`
	ByPassRLS   bool `column:"rolbypassrls" token:"BYPASSRLS"`
	ConnLimit   int  `column:"rolconnlimit" token:"CONNECTION LIMIT"`
}

var instanceRoleColumns map[string]bool

func ProcessRoleColumns(columns []string, super bool) {
	instanceRoleColumns = make(map[string]bool)
	t := reflect.TypeOf(RoleOptions{})
	for _, f := range reflect.VisibleFields(t) {
		instanceRoleColumns[f.Tag.Get("column")] = false
	}
	for _, name := range columns {
		if !super && ("rolsuper" == name || "rolreplication" == name || "rolbypassrls" == name) {
			slog.Debug("Ignoring privileged role column", "column", name)
			continue
		}
		instanceRoleColumns[name] = true
	}
}

func (o RoleOptions) String() string {
	v := reflect.ValueOf(o)
	t := v.Type()
	var b strings.Builder
	for _, f := range reflect.VisibleFields(t) {
		if !instanceRoleColumns[f.Tag.Get("column")] {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fv := v.FieldByName(f.Name)
		switch f.Type.Kind() {
		case reflect.Bool:
			o.WriteBoolOption(&b, fv.Bool(), f.Tag.Get("token"))
		case reflect.Int:
			fmt.Fprintf(&b, "%s %d", f.Tag.Get("token"), fv.Int())
		}
	}
	return b.String()
}

func (o *RoleOptions) WriteBoolOption(b *strings.Builder, value bool, token string) {
	if !value {
		b.WriteString("NO")
	}
	b.WriteString(token)
}

func (o *RoleOptions) LoadYaml(yaml map[string]interface{}) {
	for option, value := range yaml {
		switch option {
		case "SUPERUSER":
			o.Super = value.(bool)
		case "INHERIT":
			o.Inherit = value.(bool)
		case "CREATEROLE":
			o.CreateRole = value.(bool)
		case "CREATEDB":
			o.CreateDB = value.(bool)
		case "LOGIN":
			o.CanLogin = value.(bool)
		case "REPLICATION":
			o.Replication = value.(bool)
		case "BYPASSRLS":
			o.ByPassRLS = value.(bool)
		}
	}
}