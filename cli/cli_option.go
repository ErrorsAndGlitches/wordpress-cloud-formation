package cli

import (
	"github.com/urfave/cli"
	"fmt"
)

// Options here are used to generate the cli.Flag structs. The wrapper makes it explicit what the lookup key is in the
// context.
type StringCliOption interface {
	Flag() cli.Flag
	Value(context *cli.Context) string
	IsAbsent(context *cli.Context) bool
	ExitError() error
}

// There are two ways to look up options based on whether they are a global option or part of a specific command. This
// struct provides the methods that are common to both.
type StringCliOptionImpl struct {
	ShortOpt string
	LongOpt  string
	Usage    string
}

func (flag *StringCliOptionImpl) Flag() cli.Flag {
	return cli.StringFlag{
		Name:  fmt.Sprintf("%s, %s", flag.LongOpt, flag.ShortOpt),
		Usage: flag.Usage,
	}
}

func (flag *StringCliOptionImpl) ExitError() error {
	return cli.NewExitError(fmt.Sprintf("Please provide a value for %s", flag.LongOpt), 1)
}

func (flag *StringCliOptionImpl) lookupKey() string {
	return flag.ShortOpt
}

// Options for all commands
type GlobalStringCliOption struct {
	*StringCliOptionImpl
}

func (flag *GlobalStringCliOption) IsAbsent(context *cli.Context) bool {
	return flag.Value(context) == ""
}

func (flag *GlobalStringCliOption) Value(context *cli.Context) string {
	return context.GlobalString(flag.lookupKey())
}

func (flag *GlobalStringCliOption) ValueOrDefault(context *cli.Context, defaultVal string) string {
	if flag.IsAbsent(context) {
		return defaultVal
	}

	return flag.Value(context)
}

// Specific to commands
type CommandStringCliOption struct {
	*StringCliOptionImpl
}

func (flag *CommandStringCliOption) IsAbsent(context *cli.Context) bool {
	return flag.Value(context) == ""
}

func (flag *CommandStringCliOption) Value(context *cli.Context) string {
	return context.String(flag.lookupKey())
}

func (flag *CommandStringCliOption) ValueOrDefault(context *cli.Context, defaultVal string) string {
	if flag.IsAbsent(context) {
		return defaultVal
	}

	return flag.Value(context)
}
