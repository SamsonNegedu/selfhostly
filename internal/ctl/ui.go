package ctl

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// Prompter asks the user questions. Tests supply their own.
type Prompter interface {
	Confirm(question string, def bool) (bool, error)
	Input(question, hint, def string, secret bool) (string, error)
	Select(question string, options []string) (string, error)
}

type huhPrompter struct{}

func (huhPrompter) Confirm(q string, def bool) (bool, error) {
	v := def
	err := huh.NewConfirm().Title(q).Value(&v).Run()
	return v, err
}

func (huhPrompter) Input(q, hint, def string, secret bool) (string, error) {
	v := def
	in := huh.NewInput().Title(q).Value(&v)
	if hint != "" {
		in = in.Description(hint)
	}
	if secret {
		in = in.EchoMode(huh.EchoModePassword)
	}
	err := in.Run()
	return strings.TrimSpace(v), err
}

func (huhPrompter) Select(q string, options []string) (string, error) {
	var v string
	err := huh.NewSelect[string]().Title(q).Options(huh.NewOptions(options...)...).Value(&v).Run()
	return v, err
}

// interactive: questions may be asked. Never true under --non-interactive or without a terminal.
func (a *App) interactive() bool {
	if a.NonInteractive {
		return false
	}
	return a.Prompt != nil || (term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())))
}

func (a *App) prompter() Prompter {
	if a.Prompt != nil {
		return a.Prompt
	}
	return huhPrompter{}
}

// confirm answers yes under --yes, the default when nobody can be asked, and otherwise asks
func (a *App) confirm(q string, def bool) bool {
	if a.Yes {
		return true
	}
	if !a.interactive() {
		return def
	}
	v, err := a.prompter().Confirm(q, def)
	return err == nil && v
}

// ask fills *v when it is empty: from the default when nobody can be asked, else by asking.
// It fails when there is neither a value nor a default.
func (a *App) ask(v *string, q, hint, def string, secret bool) error {
	if *v != "" {
		return nil
	}
	if a.interactive() {
		ans, err := a.prompter().Input(q, hint, def, secret)
		if err != nil {
			return err
		}
		*v = ans
	} else {
		*v = def
	}
	if *v == "" && !secret {
		return fmt.Errorf("missing a value for: %s (pass it as an option, or run interactively)", q)
	}
	return nil
}

func (a *App) say(format string, args ...any) { fmt.Fprintf(a.Out, format+"\n", args...) }
func (a *App) step(format string, args ...any) {
	fmt.Fprintf(a.Out, "\n== %s ==\n", fmt.Sprintf(format, args...))
}
func (a *App) why(s string) { fmt.Fprintln(a.Out, s) }

// sleep is replaceable so tests do not wait
func (a *App) sleep(d time.Duration) {
	if a.Sleep != nil {
		a.Sleep(d)
		return
	}
	time.Sleep(d)
}
