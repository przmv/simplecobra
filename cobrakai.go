package cobrakai

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type root struct {
	c *Commandeer
}

func (r *root) Execute(ctx context.Context, args []string) (*Commandeer, error) {
	r.c.CobraCommand.SetArgs(args)
	cobraCommand, err := r.c.CobraCommand.ExecuteContextC(ctx)
	if err != nil {
		return nil, err
	}
	// Find the commandeer that was executed.
	var find func(*cobra.Command, *Commandeer) *Commandeer
	find = func(what *cobra.Command, in *Commandeer) *Commandeer {
		if in.CobraCommand == what {
			return in
		}
		for _, in2 := range in.commandeers {
			if found := find(what, in2); found != nil {
				return found
			}
		}
		return nil
	}
	return find(cobraCommand, r.c), nil
}

// Commandeer holds the state of a command and its subcommands.
type Commandeer struct {
	Command     Commander
	commandeers []*Commandeer

	// compiled
	CobraCommand *cobra.Command
}

func (c *Commandeer) compile() error {
	c.CobraCommand = &cobra.Command{
		Use:   c.Command.Name(),
		Short: "TODO",
		Long:  "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Command.Run(cmd.Context(), args)
		},
	}
	// THere's a LocalFlags set in Cobra which one would beliee would be the right place to put these flags,
	// but theat doesn't work and there's several related open issues.
	// This is how the docs say to do it and also where Hugo puts local flags.
	c.Command.AddFlagsLocal(c.CobraCommand.Flags())
	c.Command.AddFlagsPersistent(c.CobraCommand.PersistentFlags())

	// Add commands recursively.
	for _, cc := range c.commandeers {
		if err := cc.compile(); err != nil {
			return err
		}
		c.CobraCommand.AddCommand(cc.CobraCommand)
	}

	return nil
}

// Executer is the execution entry point.
// The args are usually filled with os.Args[1:].
type Executer interface {
	Execute(ctx context.Context, args []string) (*Commandeer, error)
}

// Commander is the interface that must be implemented by all commands.
type Commander interface {
	Name() string
	Run(ctx context.Context, args []string) error
	AddFlagsLocal(*pflag.FlagSet)
	AddFlagsPersistent(*pflag.FlagSet)
}

// WithCommandeer allows chaining of commandeers.
type WithCommandeer func(*Commandeer)

// R creates the execution entry poing given a root command and a chain of nested commands.
func R(command Commander, wcs ...WithCommandeer) (Executer, error) {
	c := &Commandeer{
		Command: command,
	}
	for _, wc := range wcs {
		wc(c)
	}
	if err := c.compile(); err != nil {
		return nil, err
	}
	return &root{c: c}, nil
}

// C creates nested commands.
func C(command Commander, wcs ...WithCommandeer) func(*Commandeer) {
	return func(parent *Commandeer) {
		cd := &Commandeer{
			Command: command,
		}
		parent.commandeers = append(parent.commandeers, cd)
		for _, wc := range wcs {
			wc(cd)
		}
	}
}