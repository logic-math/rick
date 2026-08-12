package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/agent/piagent"
)

// knownTheme maps a selectable theme name to the npm package that provides it
// ("" = pi built-in). rick manages the theme inside its own agent dir, so the
// choices listed here are exactly the ones rick can guarantee (it installs the
// providing package into ~/.rick/pi on demand). Custom themes dropped into
// ~/.rick/pi/agent/themes/*.json are discovered and listed as well.
type knownTheme struct {
	name string
	pkg  string
}

var knownThemes = []knownTheme{
	{name: "dark", pkg: ""},
	{name: "light", pkg: ""},
	{name: "tokyo-night-dark", pkg: "@wishx127/pi-tokyo-night"},
	{name: "tokyo-night-light", pkg: "@wishx127/pi-tokyo-night"},
	{name: "nightowl", pkg: "mitsupi"},
}

// NewThemeCmd creates the `rick tools theme` subcommand: list available pi TUI
// themes (with the active one marked) or activate one by name. Activation
// auto-installs the theme's npm package into rick's managed pi agent dir
// (~/.rick/pi/agent), keeping rick's pi runtime self-contained.
func NewThemeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "theme [name]",
		Short: "List or switch the pi TUI theme",
		Long: `List available pi TUI themes or activate one by name.

With no arguments, lists every selectable theme (pi built-ins, themes from
packages rick knows, and custom themes in ~/.rick/pi/agent/themes/*.json) and
marks the active one.

With a theme name, activates it: the providing npm package is auto-installed
into rick's managed pi agent dir if missing, then the theme is written to the
managed settings.json (hideThinkingBlock and all other settings are preserved).

Examples:
  rick tools theme            # list themes
  rick tools theme tokyo-night-light
  rick tools theme nightowl`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runThemeList(cmd.OutOrStdout())
			}
			if err := runThemeSet(args[0], cmd.OutOrStdout()); err != nil {
				return fmt.Errorf("set theme %s: %w", args[0], err)
			}
			return nil
		},
	}
}

// runThemeList prints the active theme and every selectable theme with its
// install state. Exit code 0 regardless (listing is informational).
func runThemeList(w io.Writer) error {
	if err := piagent.EnsureAgentDir(); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}
	cur := currentTheme()
	if cur == "" {
		cur = "(none — pi default)"
	}
	fmt.Fprintf(w, "Current theme: %s\n\n", cur)
	fmt.Fprintln(w, "Available themes:")
	for _, t := range knownThemes {
		state := "✓"
		if t.pkg != "" && !piListContains(filepath.Base(t.pkg)) {
			state = "not installed"
		}
		marker := " "
		if cur == t.name {
			marker = "→"
		}
		fmt.Fprintf(w, "  %s %-20s %s\n", marker, t.name, state)
	}
	for _, name := range customThemeNames() {
		marker := " "
		if cur == name {
			marker = "→"
		}
		fmt.Fprintf(w, "  %s %-20s custom (~/.rick/pi/agent/themes)\n", marker, name)
	}
	fmt.Fprintln(w, "\nSet one with: rick tools theme <name>")
	fmt.Fprintln(w, "Custom themes: drop <name>.json into ~/.rick/pi/agent/themes/")
	return nil
}

// customThemeNames lists theme names from rick's managed themes dir
// (~/.rick/pi/agent/themes/*.json), which pi discovers automatically.
func customThemeNames() []string {
	dir := filepath.Join(piagent.AgentDir(), "themes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names
}

// runThemeSet activates the named theme: resolves it (known or custom),
// auto-installs the providing npm package into the managed agent dir when
// needed, then writes the managed settings.json. Settings.json is bootstrapped
// first if missing, so hideThinkingBlock stays managed even on a fresh box.
func runThemeSet(name string, w io.Writer) error {
	pkg := ""
	known := false
	for _, t := range knownThemes {
		if t.name == name {
			pkg = t.pkg
			known = true
			break
		}
	}
	custom := false
	for _, n := range customThemeNames() {
		if n == name {
			custom = true
			break
		}
	}
	if !known && !custom {
		var choices []string
		for _, t := range knownThemes {
			choices = append(choices, t.name)
		}
		choices = append(choices, customThemeNames()...)
		return fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(choices, ", "))
	}

	// Make sure the managed settings.json exists first (bootstrap adds
	// hideThinkingBlock=true), then the theme package, then activate.
	if err := bootstrapAgentSettings(); err != nil {
		return err
	}
	if pkg != "" {
		if !piListContains(filepath.Base(pkg)) {
			fmt.Fprintf(w, "⚠️  installing theme package %s ...\n", pkg)
			piCmd := piCommand("install", "npm:"+pkg)
			piCmd.Stdout = os.Stdout
			piCmd.Stderr = os.Stderr
			if err := piCmd.Run(); err != nil {
				return fmt.Errorf("pi install npm:%s: %w", pkg, err)
			}
			if !piListContains(filepath.Base(pkg)) {
				return fmt.Errorf("%s still not listed after install", pkg)
			}
		}
	}
	if err := setTheme(name); err != nil {
		return err
	}
	if got := currentTheme(); got != name {
		return fmt.Errorf("verification failed: theme is %q, want %q", got, name)
	}
	fmt.Fprintf(w, "✅ theme active: %s\n", name)
	return nil
}
