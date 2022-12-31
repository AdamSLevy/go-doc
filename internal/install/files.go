package install

import (
	"embed"
	"fmt"
)

//go:embed all:assets
var assets embed.FS

const (
	assetsDir = "assets/"
	pluginDir = assetsDir + "plugin/"
)

var files = []FileSpec{{
	Name:            "Zsh completion script",
	FileName:        "_golang",
	AssetDir:        pluginDir,
	InstallDir:      pluginInstallDir,
	RequiredEnvVars: []RequiredEnvVar{zshCustom},
	Info: `
The Zsh completion script provides advanced completion for 'go doc' and all
other sub-commands.

It should be named '_golang' and can be placed in any directory in your FPATH.
The Zsh completion system must also be enabled, of course.

For Oh My Zsh users, it can be placed in $ZSH/custom/plugins/go/ alongside the
Oh My Zsh plugin file go.plugin.zsh.
`,
}, {
	Name:            "Oh My Zsh plugin file",
	FileName:        "go.plugin.zsh",
	AssetDir:        pluginDir,
	InstallDir:      pluginInstallDir,
	RequiredEnvVars: []RequiredEnvVar{zshCustom},
	Info: `
The Oh My Zsh plugin file defines the plugin and adds a number of go command
aliases. This is only required for Oh My Zsh users.

For Oh My Zsh to recognize the plugin, there must be a file named go.plugin.zsh
in $ZSH_CUSTOM/plugins/go/ alongside the _golang Zsh completion script. If you
don't want the aliases, the file can be empty.

You must also enable the plugin in your ~/.zshrc file:

  # ~/.zshrc
  plugins=(
    # ... your other plugins ...
    go
  )
`,
}}

var pluginInstallDir = "${ZSH_CUSTOM}/plugins/go/"
var zshCustom = RequiredEnvVar{
	Name: "ZSH_CUSTOM",
	Err:  fmt.Errorf("ZSH_CUSTOM is not set, indicating that Oh My Zsh may not be installed"),
}
