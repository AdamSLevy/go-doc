package install

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/ioutil"
)

// FileSpec is a specification for a file to be installed.
type FileSpec struct {
	// Name is the colloquial name for the file.
	//
	// This is how a human would succinctly describe it, not the file name.
	Name string
	// Info is a long form description of the file.
	//
	// It is displayed directly, so it should be pre-formatted for
	// a terminal, and start with a newline and end with two newlines.
	Info string

	// FileName is the name of the file in the embedded file system.
	//
	// It is also the name of the installed file, if InstallName is empty.
	//
	// It is joined with AssetDir to form the full path to the file in the
	// embedded file system.
	FileName string
	// InstallName is the name of the installed file.
	//
	// If empty, it is set to FileName.
	//
	// If set to "-", the file is written to stdout and not installed.
	//
	// If InstallName contains a slash or backslash, it is treated as the
	// full install path. Otherwise it is joined with InstallDir to form
	// the full path to the file in the actual file system.
	InstallName string

	// AssetDir is the embedded file system directory containing the file.
	//
	// It is joined with FileName to form the full path to the file.
	//
	// It must use the forward slash / as the path separator regardless of
	// the OS.
	//
	// It must not start with a slash.
	AssetDir string
	// InstallDir is the directory on the actual file system where the file
	// will be installed.
	//
	// It is joined with the InstallName or FileName to form the full path.
	//
	// It should use the forward slash / as the path separator regardless
	// of OS, as it will be converted to the OS's separator.
	//
	// It may include environment variables in the form $VAR or ${VAR}
	// which are expanded by os.ExpandEnv. Unset or empty variables are
	// replaced with the empty string.
	InstallDir string
	// RequiredEnvVars is a list of environment variables that must not be
	// empty for installation to proceed.
	RequiredEnvVars []RequiredEnvVar
}

// RequiredEnvVar is an environment variable that must not be empty for
// FileSpec.Install to succeed.
type RequiredEnvVar struct {
	// Name of the environment variable.
	Name string
	// If Err is not nil, FileSpec.Install will return this error if the
	// environment variable isn't set.
	Err error
}

// Check returns a nil error if the environment variable named r.Name is not
// empty.
//
// Otherwise it return r.Err if not nil, or a generic error.
func (r RequiredEnvVar) Check() error {
	if v := os.Getenv(r.Name); v != "" {
		return nil
	}
	if r.Err != nil {
		return r.Err
	}
	return fmt.Errorf("required environment variable $%s is empty", r.Name)
}

func (spec FileSpec) OpenAssetFile(assets embed.FS) (io.ReadCloser, error) {
	assetPath := path.Join(spec.AssetDir, spec.FileName)
	return assets.Open(assetPath)
}

func (spec FileSpec) Install() error {
	installPath, err := spec.installPath()
	if err != nil {
		return err
	}
	install, err := spec.confirmInstall(installPath)
	if !install {
		return err
	}
	return spec.install(installPath)
}

func (spec FileSpec) install(installPath string) error {
	if installPath == "-" {
		log.Printf("Writing the %s file to stdout", spec.Name)
	}

	install, err := stdoutOrFile(installPath)
	if err != nil {
		return err
	}
	defer install.Close()

	asset, err := spec.OpenAssetFile(assets)
	if err != nil {
		return err
	}
	defer asset.Close()

	_, err = io.Copy(install, asset)
	log.Println()
	return err
}

func stdoutOrFile(installPath string) (io.WriteCloser, error) {
	if installPath == "-" {
		return ioutil.WriteNopCloser(os.Stdout), nil
	}
	if err := os.Mkdir(filepath.Dir(installPath), 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}

	return os.Create(installPath)
}

func (spec FileSpec) confirmInstall(installPath string) (install bool, _ error) {
	log.Println()
	prompt := fmt.Sprintf("Install the %s file to %s? [y/N/p/?]", spec.Name, installPath)
	for i := 0; i < 2; i++ {
		response, err := askUser(prompt)
		if err != nil {
			return false, err
		}
		response = strings.TrimSpace(response)
		response = strings.ToLower(response)
		switch response {
		case "y", "yes":
			return true, nil
		case "p":
			return false, spec.install("-")
		case "?":
			log.Println(spec.Info)
			log.Println("use 'y' to install, 'n' to skip, 'p' to print the file to stdout, or '?' to show this message again")
			continue
		}
		break
	}
	return false, fmt.Errorf("installation of %s skipped by user", spec.Name)
}
func askUser(prompt string) (response string, _ error) {
	log.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

func (spec FileSpec) installPath() (string, error) {
	installDir := spec.InstallDir
	for _, envVar := range spec.RequiredEnvVars {
		if err := envVar.Check(); err != nil {
			return "", err
		}
	}
	installDir = os.ExpandEnv(spec.InstallDir)
	installDir = filepath.FromSlash(installDir)

	installName := spec.InstallName
	if installName == "" {
		// InstallName is the same as FileName.
		installName = spec.FileName
	}

	installPath := filepath.Join(installDir, installName)
	installPath = filepath.Clean(installPath)
	return installPath, nil
}

//go:embed all:assets
var assets embed.FS

const (
	assetsDir = "assets/"
	pluginDir = assetsDir + "plugin/"
	binDir    = assetsDir + "bin/"
)

var files = []FileSpec{{
	Name:       "Go command shim script",
	FileName:   "go",
	AssetDir:   binDir,
	InstallDir: "${HOME}/bin/",
	RequiredEnvVars: []RequiredEnvVar{{
		Name: "HOME",
		Err:  errors.New("HOME environment variable is not set, so cannot install the go command shim script to ${HOME}/bin/"),
	}},
	Info: `
The Go command shim script allows for using go-doc when you type 'go doc' on
the command line.

It must be named 'go' and placed in a directory that occurs in your PATH ahead
of the directory where the real go binary is.

For all sub-commands except 'doc', the shim script calls the next go executable
that occurs in your PATH, with the original arguments.

For the 'doc' subcommand, the shim script calls go-doc with the original
arguments after 'doc'. 

If 'go - doc' is called, the shim script will call the real go binary without
the original arguments except the leading '-'.
`,
}, {
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
