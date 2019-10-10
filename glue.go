package glue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

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

	// app is implementation of App.
	app struct {
		ctx      context.Context
		mux      sync.Mutex
		scopes   []string
		bundles  map[string]Bundle
		builder  *di.Builder
		registry Registry
	}

	// optionFunc wraps a func so it satisfies the Option interface.
	optionFunc func(kernel *app) error
)

const (
	// DefCliRoot is root command definition name.
	DefCliRoot = "cli.cmd.root"

	// DefCliVersion is version command definition name.
	DefCliVersion = "cli.cmd.version"

	// DefContext is global context definition name.
	DefContext = "context"

	// DefRegistry is registry definition name.
	// Deprecated: use Context instead of Registry. Will be removed in 3.0.
	DefRegistry = "registry"

	// TagCliCommand is tag to mark exported cli commands.
	TagCliCommand = "cli.cmd"

	// TagRootPersistentFlags is tag to mark FlagSet for add to root command persistent flags
	TagRootPersistentFlags = "cli.persistent_flags"
)

// ErrNilContext is error triggered when detected nil context in option value.
var ErrNilContext = errors.New("nil context")

// Context option.
func Context(ctx context.Context) Option {
	return optionFunc(func(a *app) error {
		if ctx == nil {
			return ErrNilContext
		}

		a.ctx = ctx
		return nil
	})
}

// Bundles option.
func Bundles(bundles ...Bundle) Option {
	return optionFunc(func(a *app) error {
		for _, bundle := range bundles {
			if _, ok := a.bundles[bundle.Name()]; ok {
				return fmt.Errorf(`trying to register two bundles with the same name "%s"`, bundle.Name())
			}

			a.bundles[bundle.Name()] = bundle
		}

		return nil
	})
}

// Scopes option.
func Scopes(scopes ...string) Option {
	return optionFunc(func(a *app) error {
		a.scopes = scopes
		return nil
	})
}

// Version option.
func Version(version string) Option {
	return optionFunc(func(a *app) error {
		a.withValue("app.version", version)
		return nil
	})
}

// NewApp is app constructor.
func NewApp(options ...Option) (_ App, err error) {
	var a = app{
		ctx:      context.Background(),
		bundles:  make(map[string]Bundle, 8),
		registry: newRegistry(),
	}

	// apply options
	for _, option := range options {
		if err = option.apply(&a); err != nil {
			return nil, err
		}
	}

	// create di builder
	if a.builder, err = di.NewBuilder(a.scopes...); err != nil {
		return nil, err
	}

	// initialize builder
	if err = a.initBuilder(); err != nil {
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

	// modify context
	var cancelFunc = a.withCancel()
	a.withValue("app.path", appPath)

	// build container
	var container = a.builder.Build()
	defer func() {
		cancelFunc()

		if err != nil {
			_ = container.Delete()
			return
		}

		err = container.Delete()
	}()

	// wait signal, cancel execution context
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			cancelFunc()
		}
	}()

	// resolve cli root
	var root *cobra.Command
	if err = container.Fill(DefCliRoot, &root); err != nil {
		return err
	}

	// TODO: Pass context, when PR will merged. See: https://github.com/spf13/cobra/pull/893
	err = root.Execute()
	return
}

// initBuilder initialize di builder
func (a *app) initBuilder() error {
	return a.builder.Add(
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
						a.withValue("cli.cmd", cmd)
						a.withValue("cli.args", args)
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
				var ctx context.Context
				if err = ctn.Fill(DefContext, &ctx); err != nil {
					return nil, err
				}

				return &cobra.Command{
					Use:           "version",
					Short:         "Application version",
					SilenceUsage:  true,
					SilenceErrors: true,
					Run: func(cmd *cobra.Command, args []string) {
						if v, ok := ctx.Value("app.version").(string); ok {
							fmt.Println(v)
						}
					},
				}, nil
			},
		},
		di.Def{
			Name: DefContext,
			Build: func(_ di.Container) (interface{}, error) {
				return a.ctx, nil
			},
			Unshared: true,
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
		if err = a.bundles[name].Build(a.builder); err != nil {
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

// withCancel append cancellation to current context. Method is non thread safe.
func (a *app) withCancel() (fn context.CancelFunc) {
	a.ctx, fn = context.WithCancel(a.ctx)
	return fn
}

// withValue append any value to current context. Method is non thread safe.
func (a *app) withValue(key, value interface{}) context.Context {
	a.ctx = context.WithValue(a.ctx, key, value)
	return a.ctx
}

// apply implements Option.
func (f optionFunc) apply(kernel *app) error {
	return f(kernel)
}
