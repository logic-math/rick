package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sunquan/rick/internal/env"
	"github.com/sunquan/rick/internal/runtime"
)

// knownTheme maps a selectable theme name to the npm package that provides it
// ("" = pi built-in; embedded = rick-shipped custom theme file written into the
// managed themes dir on activation). rick manages the theme inside its own
// agent dir, so the choices listed here are exactly the ones rick can guarantee
// (it installs the providing package into ~/.rick/pi on demand). Custom themes
// dropped into ~/.rick/pi/agent/themes/*.json are discovered and listed too.
//
// The embedded theme JSON lives in internal/env/themes/ and is written by
// env.WriteEmbeddedTheme; this struct only carries the embed path mapping.
type knownTheme struct {
	name     string
	pkg      string
	embedded string // embedded themes/<file>.json, written to the managed themes dir on set
}

var knownThemes = []knownTheme{
	// rick 默认主题（内置，基于 GitHub Dark Dimmed 定制）：工具标题/命令绿、
	// md 标题金、链接/路径蓝、bashMode 金 —— AI 正式回复最突出。
	{name: "rick", embedded: "themes/rick.json"},
	{name: "dark", pkg: ""},
	{name: "light", pkg: ""},
	// Night Owl（Armin 的 pi 包）
	{name: "nightowl", pkg: "mitsupi"},
	// Jellybeans Mono（暗/亮）
	{name: "jellybeans-mono", pkg: "@aliou/pi-theme-jellybeans"},
	{name: "jellybeans-mono-light", pkg: "@aliou/pi-theme-jellybeans"},
	// Gruber Darker / Lighter
	{name: "gruber-darker", pkg: "pi-theme-gruber-darker"},
	{name: "gruber-lighter", pkg: "pi-theme-gruber-darker"},
	// Cyberpunk 高对比（4 色）
	{name: "ameno-cyberdyne", pkg: "ameno-cyberdyne"},
	{name: "ameno-cyberdyne-teal", pkg: "ameno-cyberdyne"},
	{name: "ameno-cyberdyne-blue", pkg: "ameno-cyberdyne"},
	{name: "ameno-cyberdyne-soft", pkg: "ameno-cyberdyne"},
	// Poimandres（VSCode 风格）
	{name: "poimandres", pkg: "@llttlltt/poimandres-pi"},
	{name: "poimandres-storm", pkg: "@llttlltt/poimandres-pi"},
	{name: "poimandres-white", pkg: "@llttlltt/poimandres-pi"},
	// GitHub 风格（rick 内置，基于 GitHub Primer 配色）
	{name: "gh-dark", pkg: "pi-gh-dark-theme"},
	{name: "gh-dark-dimmed", embedded: "themes/gh-dark-dimmed.json"},
	{name: "gh-light", embedded: "themes/gh-light.json"},
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
  rick tools theme gh-light
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
	if err := runtime.EnsureAgentDir(); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}
	cur := env.CurrentTheme()
	if cur == "" {
		cur = "(none — pi default)"
	}
	fmt.Fprintf(w, "Current theme: %s\n\n", cur)
	fmt.Fprintln(w, "Available themes:")
	for _, t := range knownThemes {
		state := "✓"
		switch {
		case t.embedded != "":
			state = "builtin (rick)"
		case t.pkg != "" && !env.PiListContains(filepath.Base(t.pkg)):
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
	dir := filepath.Join(runtime.AgentDir(), "themes")
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
	embedded := ""
	known := false
	for _, t := range knownThemes {
		if t.name == name {
			pkg = t.pkg
			embedded = t.embedded
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
	// hideThinkingBlock=true), then the theme source, then activate.
	if err := env.BootstrapAgentSettings(); err != nil {
		return err
	}
	if embedded != "" {
		// rick-shipped theme: write the embedded JSON into the managed themes
		// dir, where pi discovers it automatically (agentDir/themes/*.json).
		if err := env.WriteEmbeddedTheme(embedded); err != nil {
			return err
		}
	}
	if pkg != "" {
		if !env.PiListContains(filepath.Base(pkg)) {
			fmt.Fprintf(w, "⚠️  installing theme package %s ...\n", pkg)
			piCmd := env.PiCommand("install", "npm:"+pkg)
			piCmd.Stdout = os.Stdout
			piCmd.Stderr = os.Stderr
			if err := piCmd.Run(); err != nil {
				return fmt.Errorf("pi install npm:%s: %w", pkg, err)
			}
			if !env.PiListContains(filepath.Base(pkg)) {
				return fmt.Errorf("%s still not listed after install", pkg)
			}
		}
	}
	if err := env.SetTheme(name); err != nil {
		return err
	}
	if got := env.CurrentTheme(); got != name {
		return fmt.Errorf("verification failed: theme is %q, want %q", got, name)
	}
	fmt.Fprintf(w, "✅ theme active: %s\n", name)
	return nil
}
