// Copyright 2018 Sergey Novichkov. All rights reserved.
// For the full copyright and license information, please view the LICENSE
// file that was distributed with this source code.

package glue

import "github.com/gozix/di"

const (
	// tagCliCommand is tag to mark exported cli commands.
	tagCliCommand = "cli.cmd"

	// tagCliRootCommand is tag to mark app root cli command.
	tagCliRootCommand = "cli.cmd.root"

	// tagPersistentFlags is tag to mark FlagSet for add to root command persistent flags
	tagPersistentFlags = "cli.persistent_flags"

	// tagPersistentPreRunner is tag to mark persistent prerunners
	tagPersistentPreRunner = "cli.persistent_prerunner"
)

// AsCliCommand is syntax sugar for the di container.
func AsCliCommand() di.ProvideOption {
	return di.Tags{{
		Name: tagCliCommand,
	}}
}

// AsPersistentFlags is syntax sugar for the di container.
func AsPersistentFlags() di.ProvideOption {
	return di.Tags{{
		Name: tagPersistentFlags,
	}}
}

// AsPersistentPreRunner is syntax sugar for the di container.
func AsPersistentPreRunner() di.ProvideOption {
	return di.Tags{{
		Name: tagPersistentPreRunner,
	}}
}

func asRootCommand() di.ProvideOption {
	return di.Tags{{
		Name: tagCliRootCommand,
	}}
}

func withCliCommand() di.Modifier {
	return di.WithTags(tagCliCommand)
}

func withPersistentFlags() di.Modifier {
	return di.WithTags(tagPersistentFlags)
}

func withPersistentPreRunner() di.Modifier {
	return di.WithTags(tagPersistentPreRunner)
}

func withRootCommand() di.Modifier {
	return di.WithTags(tagCliRootCommand)
}
