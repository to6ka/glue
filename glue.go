// Package glue provide basic functionality.
package glue

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/sarulabs/di"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type (
	// App interface.
	App interface {
		Execute() error
	}

	// Option interface.
	Option interface {
		apply(kernel *app) error
	}

	// Bundle is an node interface.
	Bundle interface {
		Name() string
		Build(builder *di.Builder) error
	}

	// BundleDependsOn is an node with dependencies interface.
	BundleDependsOn interface {
		Bundle
		DependsOn() []string
	}

	// Registry interface.
	Registry interface {
		Get(name string) interface{}
		Set(name string, value interface{})
		Fill(name string, value interface{}) error
	}

	// app is implementation of App.
	app struct {
		mux       sync.Mutex
		scopes    []string
		bundles   map[string]Bundle
		registry  Registry
		container *di.Builder
	}

	// optionFunc wraps a func so it satisfies the Option interface.
	optionFunc func(kernel *app) error

	// registry is implementation of Registry.
	registry struct {
		mux    sync.RWMutex
		values map[string]interface{}
	}
)

const (
	// DefCliRoot is root command definition name.
	DefCliRoot = "cli.cmd.root"

	// DefCliVersion is version command definition name.
	DefCliVersion = "cli.cmd.version"

	// DefRegistry is registry definition name.
	DefRegistry = "registry"

	// TagCliCommand is tag to mark exported cli commands.
	TagCliCommand = "cli.cmd"

	// TagRootPersistentFlags is tag to mark FlagSet for add to root command persistent flags
	TagRootPersistentFlags = "cli.persistent_flags"
)

// Bundles option.
func Bundles(bundles ...Bundle) Option {
	return optionFunc(func(k *app) error {
		for _, bundle := range bundles {
			if _, ok := k.bundles[bundle.Name()]; ok {
				return fmt.Errorf(`trying to register two bundles with the same name "%s"`, bundle.Name())
			}

			k.bundles[bundle.Name()] = bundle
		}

		return nil
	})
}

// Scopes option.
func Scopes(scopes ...string) Option {
	return optionFunc(func(k *app) error {
		k.scopes = scopes
		return nil
	})
}

// Version option.
func Version(version string) Option {
	return optionFunc(func(k *app) error {
		k.registry.Set("app.version", version)
		return nil
	})
}

// NewApp is app constructor.
func NewApp(options ...Option) (_ App, err error) {
	var (
		p = registry{
			values: make(map[string]interface{}),
		}
		a = app{
			bundles:  make(map[string]Bundle, 8),
			registry: &p,
		}
	)

	// apply options
	for _, option := range options {
		if err = option.apply(&a); err != nil {
			return nil, err
		}
	}

	// create di container
	if a.container, err = di.NewBuilder(a.scopes...); err != nil {
		return nil, err
	}

	// initialize container
	if err = a.initContainer(); err != nil {
		return nil, err
	}

	// register bundles
	if err = a.registerBundles(); err != nil {
		return nil, err
	}

	return &a, nil
}

// Execute implementation.
func (a *app) Execute() (err error) {
	a.mux.Lock()
	defer a.mux.Unlock()

	// app.path
	var appPath string
	if appPath, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
		return err
	}

	a.registry.Set("app.path", appPath)

	// build container
	var context = a.container.Build()
	defer func() {
		if err != nil {
			_ = context.Delete()
			return
		}

		err = context.Delete()
	}()

	// resolve cli root
	var root *cobra.Command
	if err = context.Fill(DefCliRoot, &root); err != nil {
		return err
	}

	return root.Execute()
}

// initContainer initialize di container
func (a *app) initContainer() error {
	return a.container.Add(
		di.Def{
			Name: DefCliRoot,
			Build: func(ctn di.Container) (_ interface{}, err error) {
				var registry Registry
				if err = ctn.Fill(DefRegistry, &registry); err != nil {
					return nil, err
				}

				var rootCmd = cobra.Command{
					Use:           fmt.Sprintf("%s [command]", os.Args[0]), // TODO: replace to binary name
					SilenceUsage:  true,
					SilenceErrors: true,
					PersistentPreRun: func(cmd *cobra.Command, args []string) {
						registry.Set("cli.cmd", cmd)
						registry.Set("cli.args", args)
					},
				}

				// register commands by tag
				for name, def := range ctn.Definitions() {
				Tags:
					for _, tag := range def.Tags {
						switch tag.Name {
						case TagCliCommand:
							var command *cobra.Command
							if err = ctn.Fill(name, &command); err != nil {
								return nil, err
							}

							rootCmd.AddCommand(command)
							break Tags
						case TagRootPersistentFlags:
							var pf *pflag.FlagSet
							if err = ctn.Fill(name, &pf); err != nil {
								return nil, err
							}

							rootCmd.PersistentFlags().AddFlagSet(pf)
							break Tags
						}
					}
				}

				return &rootCmd, nil
			},
		},
		di.Def{
			Name: DefCliVersion,
			Tags: []di.Tag{{
				Name: TagCliCommand,
			}},
			Build: func(ctn di.Container) (_ interface{}, err error) {
				var p Registry
				if err = ctn.Fill(DefRegistry, &p); err != nil {
					return nil, err
				}

				return &cobra.Command{
					Use:           "version",
					Short:         "Application version",
					SilenceUsage:  true,
					SilenceErrors: true,
					Run: func(cmd *cobra.Command, args []string) {
						fmt.Println(
							p.Get("app.version"),
						)
					},
				}, nil
			},
		},
		di.Def{
			Name: DefRegistry,
			Build: func(_ di.Container) (interface{}, error) {
				return a.registry, nil
			},
		},
	)
}

// registerBundles resolve bundles dependencies and register them.
func (a *app) registerBundles() (err error) {
	// resolve dependencies
	var (
		resolved   = make([]string, 0, len(a.bundles))
		unresolved = make([]string, 0, len(a.bundles))
	)

	for _, bundle := range a.bundles {
		if err = a.resolveDependencies(bundle, &resolved, &unresolved); err != nil {
			return err
		}
	}

	// register
	for _, name := range resolved {
		if err = a.bundles[name].Build(a.container); err != nil {
			return err
		}
	}

	return nil
}

// dependencies generate dependencies graph.
func (a *app) resolveDependencies(bundle Bundle, resolved *[]string, unresolved *[]string) (err error) {
	for _, name := range *unresolved {
		if bundle.Name() == name {
			return fmt.Errorf(`"%s" has circular dependency of itself`, bundle.Name())
		}
	}

	*unresolved = append(*unresolved, bundle.Name())

	if v, ok := bundle.(BundleDependsOn); ok {
		for _, name := range v.DependsOn() {
			if v.Name() == name {
				return fmt.Errorf(`"%s" can not depends on itself`, v.Name())
			}

			if _, ok := a.bundles[name]; !ok {
				return fmt.Errorf(`"%s" has unresoled dependency "%s"`, v.Name(), name)
			}

			if err = a.resolveDependencies(a.bundles[name], resolved, unresolved); err != nil {
				return err
			}
		}
	}

	*unresolved = (*unresolved)[:len(*unresolved)-1]

	for _, name := range *resolved {
		if bundle.Name() == name {
			return nil
		}
	}

	*resolved = append(*resolved, bundle.Name())

	return nil
}

// apply implements Option.
func (f optionFunc) apply(kernel *app) error {
	return f(kernel)
}

// Get implements Registry.
func (p *registry) Get(name string) interface{} {
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.values[name]
}

// Set implements Registry.
func (p *registry) Set(name string, value interface{}) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.values[name] = value
}

// Fill implements Registry.
func (p *registry) Fill(name string, target interface{}) (err error) {
	var src interface{}
	defer func() {
		if r := recover(); r != nil {
			t := reflect.TypeOf(target)
			s := reflect.TypeOf(src)
			err = fmt.Errorf("target is `%s` but should be a pointer to the source type `%s`", t, s)
		}
	}()

	src = p.Get(name)
	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(src))
	return
}
