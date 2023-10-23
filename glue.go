// Copyright 2018 Sergey Novichkov. All rights reserved.
// For the full copyright and license information, please view the LICENSE
// file that was distributed with this source code.

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

	"github.com/gozix/di"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//go:generate mockery --case=underscore --output=mock --outpkg=mock --name=Bundle|BundleDependsOn
type (
	// App interface.
	App interface {
		Execute() error
	}

	InternalApp interface {
		Container() di.Container
		Init() error
		Run() error
		Stop()
	}

	// Option interface.
	Option interface {
		apply(kernel *app) error
	}

	// Bundle is an node interface.
	Bundle interface {
		Name() string
		Build(builder di.Builder) error
	}

	// BundleDependsOn is an node with dependencies interface.
	BundleDependsOn interface {
		Bundle
		DependsOn() []string
	}

	// PreRunner is a persistent prerunner interface.
	PreRunner interface {
		Run(ctx context.Context) error
	}

	// PreRunnerFunc is syntax sugar for usage PreRunner.
	PreRunnerFunc func(ctx context.Context) error

	// app is implementation of App.
	app struct {
		ctx       context.Context
		cancel    context.CancelFunc
		mux       sync.Mutex
		bundles   map[string]Bundle
		builder   di.Builder
		container di.Container
	}

	// optionFunc wraps a func, so it satisfies the Option interface.
	optionFunc func(kernel *app) error
)

var (
	// ErrNilContext is error triggered when detected nil context in option value.
	ErrNilContext = errors.New("nil context")

	_ PreRunner = (*PreRunnerFunc)(nil)
)

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

// Version option.
func Version(version string) Option {
	return optionFunc(func(a *app) error {
		a.withValue("app.version", version)
		return nil
	})
}

func NewApp(options ...Option) (App, error) {
	return newApp(options...)
}

func NewInternalApp(options ...Option) (InternalApp, error) {
	return newApp(options...)
}

func newApp(options ...Option) (*app, error) {
	var a = app{
		ctx:     context.Background(),
		bundles: make(map[string]Bundle, 8),
	}

	// apply options
	var err error
	for _, option := range options {
		if err = option.apply(&a); err != nil {
			return nil, err
		}
	}

	// create di builder
	if a.builder, err = a.initBuilder(); err != nil {
		return nil, err
	}

	// register bundles
	if err = a.registerBundles(); err != nil {
		return nil, err
	}

	return &a, nil
}

// Execute implementation.
func (a *app) Execute() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	// wait signal, cancel execution context
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		a.Stop()
	}()

	err := a.init()
	if err != nil {
		return err
	}

	return a.run()
}

func (a *app) Init() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	return a.init()
}

func (a *app) init() error {
	// app.path
	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("unable resolve app.path : %w", err)
	}

	// modify context
	a.withCancel()
	a.withValue("app.path", appPath)

	// build container
	a.container, err = a.builder.Build()
	if err != nil {
		return err
	}

	return nil
}

func (a *app) Run() error {
	a.mux.Lock()
	defer a.mux.Unlock()

	return a.run()
}

func (a *app) run() (err error) {
	defer func() {
		a.Stop()

		if err != nil {
			_ = a.container.Close()
			return
		}

		err = a.container.Close()
	}()

	// resolve cli root
	var root *cobra.Command
	if err = a.container.Resolve(&root, withRootCommand()); err != nil {
		return err
	}

	return root.ExecuteContext(a.ctx)
}

func (a *app) Stop() {
	a.cancel()
}

func (a *app) Container() di.Container {
	return a.container
}

// builder initialize di builder
func (a *app) initBuilder() (di.Builder, error) {
	return di.NewBuilder(
		di.Provide(a.provideRootContext, di.Unshared()),
		di.Provide(
			a.provideRootCmd,
			di.Constraint(0, di.Optional(true), withPersistentPreRunner()),
			di.Constraint(1, di.Optional(true), withPersistentFlags()),
			di.Constraint(2, di.Optional(true), withCliCommand()),
			asRootCommand(),
		),
		di.Provide(a.provideVersionCmd, AsCliCommand()),
	)
}

func (a *app) provideRootCmd(preRunners []PreRunner, flagSets []*pflag.FlagSet, subCommands []*cobra.Command) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:           fmt.Sprintf("%s [command]", os.Args[0]), // TODO: replace to binary name
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			a.withValue("cli.cmd", cmd)
			a.withValue("cli.args", args)

			for _, preRunner := range preRunners {
				if err = preRunner.Run(a.ctx); err != nil {
					return err
				}
			}

			return nil
		},
	}

	// register flagSets
	for _, flagSet := range flagSets {
		rootCmd.PersistentFlags().AddFlagSet(flagSet)
	}

	// register sub commands
	rootCmd.AddCommand(subCommands...)

	return rootCmd
}

func (a *app) provideVersionCmd(ctx context.Context) *cobra.Command {
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
	}
}

func (a *app) provideRootContext() context.Context {
	return a.ctx
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
func (a *app) withCancel() {
	a.ctx, a.cancel = context.WithCancel(a.ctx)
}

// withValue append any value to current context. Method is non thread safe.
func (a *app) withValue(key, value interface{}) context.Context {
	a.ctx = context.WithValue(a.ctx, key, value)
	return a.ctx
}

func (p PreRunnerFunc) Run(ctx context.Context) error {
	return p(ctx)
}

// apply implements Option.
func (f optionFunc) apply(kernel *app) error {
	return f(kernel)
}
