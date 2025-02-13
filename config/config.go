package config

import (
	"bruce/loader"
	"bruce/operators"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

// TemplateData will be marshalled from the provided config file that exists.
type TemplateData struct {
	Steps     []Steps           `yaml:"steps"`
	Variables map[string]string `yaml:"variables"`
	BackupDir string
}

// Steps include multiple action operators to be executed per step
type Steps struct {
	Name   string             `yaml:"name"`
	Action operators.Operator `yaml:"action"`
}

// TODO: Add UnmarshallJSON

// UnmarshalYAML Implements the Unmarshaler interface of the yaml pkg.
func (e *Steps) UnmarshalYAML(nd *yaml.Node) error {
	// TODO: Fix this is the near future. (maybe plugin based?)

	crn := &operators.Cron{}
	if err := nd.Decode(crn); err == nil && len(crn.Schedule) > 0 {
		log.Debug().Msg("matching cron operator")
		e.Action = crn
		return nil
	}

	co := &operators.Command{}
	if err := nd.Decode(co); err == nil && len(co.Cmd) > 0 {
		log.Debug().Msg("matching command operator")
		e.Action = co
		return nil
	}

	tb := &operators.Tarball{}
	if err := nd.Decode(tb); err == nil && len(tb.Src) > 0 {
		log.Debug().Msg("matching tarball operator")
		e.Action = tb
		return nil
	}

	cp := &operators.Copy{}
	if err := nd.Decode(cp); err == nil && len(cp.Src) > 0 {
		log.Debug().Msg("matching copy operator")
		e.Action = cp
		return nil
	}

	to := &operators.Template{}
	if err := nd.Decode(to); err == nil && len(to.Template) > 0 {
		log.Debug().Msg("matching template operator")
		e.Action = to
		return nil
	}

	gt := &operators.Git{}
	if err := nd.Decode(gt); err == nil && len(gt.Repo) > 0 {
		log.Debug().Msg("matching git operator")
		e.Action = gt
		return nil
	}

	rc := &operators.RecursiveCopy{}
	if err := nd.Decode(rc); err == nil && len(rc.Src) > 0 {
		log.Debug().Msg("matching recursive copy operator")
		e.Action = rc
		return nil
	}

	lp := &operators.Loop{}
	if err := nd.Decode(lp); err == nil && len(lp.LoopScript) > 0 {
		log.Debug().Msg("matching loop operator")
		e.Action = lp
		return nil
	}
	re := &operators.RemoteExec{}
	if err := nd.Decode(re); err == nil && len(re.ExecCmd) > 0 {
		log.Debug().Msg("matching remote exec operator")
		e.Action = re
		return nil
	}
	ap := &operators.API{}
	if err := nd.Decode(ap); err == nil && len(ap.Endpoint) > 0 {
		log.Debug().Msg("matching api operator")
		e.Action = ap
		return nil
	}
	slp := &operators.Sleep{}
	if err := nd.Decode(slp); err == nil && slp.Time > 0 {
		log.Debug().Msg("matching sleep operator")
		e.Action = slp
		return nil
	}

	log.Debug().Msg("no matching operator found, using null operator")
	e.Action = &operators.NullOperator{}
	return nil
}

// LoadConfig attempts to load the user provided manifest.
func LoadConfig(fileName, key string) (*TemplateData, error) {
	d, _, err := loader.ReadRemoteFile(fileName, key)
	if err != nil {
		log.Error().Err(err).Msg("cannot proceed without a config file and specified config cannot be read.")
		os.Exit(1)
	}
	log.Debug().Bytes("rawConfig", d)
	c := &TemplateData{}

	if os.Getenv("BRUCE_DEBUG") == "true" {
		log.Debug().Msg("debug mode enabled")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	// String replace all template variable braces {{ with --== and }} with ==-- in d to avoid yaml parsing issues
	d = []byte(strings.ReplaceAll(strings.ReplaceAll(string(d), "{{", "--=="), "}}", "==--"))

	err = yaml.Unmarshal(d, c)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse config file")
	}

	return c, nil
}
