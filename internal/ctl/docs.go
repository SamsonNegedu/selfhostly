package ctl

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// docsCmd writes the command reference from the command definitions themselves, so the reference cannot
// drift from what the tool does. It is hidden: it is for maintainers (`make docs`).
func (a *App) docsCmd(root *cobra.Command) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Write the command reference as Markdown",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			md := ReferenceMarkdown(root)
			if out == "" {
				_, err := fmt.Fprint(a.Out, md)
				return err
			}
			return os.WriteFile(out, []byte(md), 0o644)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "write to this file instead of standard output")
	return cmd
}

// ReferenceMarkdown renders every visible command, with its flags, as one Markdown page
func ReferenceMarkdown(root *cobra.Command) string {
	var b strings.Builder
	b.WriteString("# selfhostlyctl reference\n\n")
	b.WriteString("<!-- Generated from the command definitions by `make docs`. Do not edit by hand. -->\n\n")
	b.WriteString(root.Short + ". Every command explains itself with `--help`.\n\n")

	cmds := visibleCommands(root)
	b.WriteString("| Command | What it does |\n|---|---|\n")
	for _, c := range cmds {
		fmt.Fprintf(&b, "| [`%s`](#%s) | %s |\n", c.CommandPath(), anchor(c.CommandPath()), c.Short)
	}

	b.WriteString("\n## Options for every command\n\n")
	writeFlags(&b, root.PersistentFlags())

	for _, c := range cmds {
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n```\n%s\n```\n", c.CommandPath(), c.Short, c.UseLine())
		if c.Long != "" {
			fmt.Fprintf(&b, "\n```\n%s\n```\n", strings.TrimSpace(c.Long))
		}
		if c.LocalNonPersistentFlags().HasAvailableFlags() {
			b.WriteString("\n")
			writeFlags(&b, c.LocalNonPersistentFlags())
		}
	}
	return b.String()
}

// visibleCommands returns the commands and subcommands a person can run, in name order
func visibleCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		kids := append([]*cobra.Command(nil), c.Commands()...)
		sort.Slice(kids, func(i, j int) bool { return kids[i].Name() < kids[j].Name() })
		for _, k := range kids {
			if k.Hidden || k.Name() == "help" || k.Name() == "completion" {
				continue
			}
			if k.Runnable() {
				out = append(out, k)
			}
			walk(k)
		}
	}
	walk(root)
	return out
}

func writeFlags(b *strings.Builder, fs *pflag.FlagSet) {
	b.WriteString("| Option | Meaning |\n|---|---|\n")
	fs.SortFlags = true
	fs.VisitAll(func(f *pflag.Flag) {
		name := "--" + f.Name
		if f.Shorthand != "" {
			name = "-" + f.Shorthand + ", " + name
		}
		if t := f.Value.Type(); t != "bool" {
			name += " " + t
		}
		desc := f.Usage
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			desc += fmt.Sprintf(" (default %s)", f.DefValue)
		}
		fmt.Fprintf(b, "| `%s` | %s |\n", name, desc)
	})
}

func anchor(path string) string {
	return strings.ReplaceAll(strings.ToLower(path), " ", "-")
}
