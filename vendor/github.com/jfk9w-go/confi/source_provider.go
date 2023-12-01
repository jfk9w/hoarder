package confi

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type SourceProvider interface {
	GetSources(ctx context.Context) ([]Source, error)
}

type DefaultSourceProvider struct {
	EnvPrefix string
	Env       []string
	Args      []string
	Stdin     io.Reader
}

func (p *DefaultSourceProvider) GetSources(ctx context.Context) ([]Source, error) {
	var envProps, argProps []Property
	for _, env := range p.Env {
		if strings.HasPrefix(env, p.EnvPrefix) {
			env = env[len(p.EnvPrefix):]
			prop, err := getProperty(env, "_", false)
			if err != nil {
				return nil, errors.Wrapf(err, `env "%s"`, env)
			}

			envProps = append(envProps, *prop)
		}
	}

	for _, arg := range p.Args {
		if strings.HasPrefix(arg, "--") {
			arg = arg[2:]
			prop, err := getProperty(arg, ".", true)
			if err != nil {
				return nil, errors.Wrapf(err, `arg "%s"`, arg)
			}

			argProps = append(argProps, *prop)
		}
	}

	var (
		envs  PropertySource
		files []Source
		stdin *InputSource
		args  PropertySource
	)

	for _, item := range []struct {
		source *PropertySource
		props  []Property
	}{
		{source: &envs, props: envProps},
		{source: &args, props: argProps},
	} {
		hasFiles := false
		for _, prop := range item.props {
			switch prop.Key() {
			case "config.file":
				if !hasFiles {
					files = make([]Source, 0)
				}

				format := filepath.Ext(prop.Value)
				if format == "" {
					continue
				}

				files = append(files, InputSource{
					Input:  File(prop.Value),
					Format: format[1:],
				})

				hasFiles = true

			case "config.stdin":
				if prop.Value == "" {
					stdin = nil
				} else {
					stdin = &InputSource{
						Input:  Reader{R: p.Stdin},
						Format: prop.Value,
					}
				}

			default:
				*item.source = append(*item.source, prop)
			}
		}
	}

	sources := make([]Source, 0)
	if envs != nil {
		sources = append(sources, envs)
	}

	sources = append(sources, files...)
	if stdin != nil {
		sources = append(sources, *stdin)
	}

	if args != nil {
		sources = append(sources, args)
	}

	return sources, nil
}
