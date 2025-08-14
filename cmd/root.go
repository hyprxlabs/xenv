/*
Copyright © 2025 hyprxdev <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hyprxlabs/go/cmdargs"
	"github.com/hyprxlabs/go/dotenv"
	"github.com/hyprxlabs/go/env"
	"github.com/hyprxlabs/go/exec"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xenv [OPTIONS] [NAME=VALUE...] [ENV_FILE...] [COMMAND] [ARG...]",
	Short: "xenv sets environment variables and executes a command",
	Long: `xenv sets environment variables from the command line, .env files, or both,
and executes a command with those environment variables set.

If no command is provided, the environment variables are printed (unless --quiet is passed in).

Variable expansion is supported in the following formats:
- $VAR: Expands to the value of the environment variable VAR.
- ${VAR}: Same as above, but allows for more complex expressions.
- ${VAR:-default}: If VAR is unset or null, expands to default.
- ${VAR:=default}: If VAR is unset or null, sets VAR to default and expands to default.
- ${VAR:?error}: If VAR is unset or null, prints error and exits.

Command substitution is supported with the $(command) syntax, which executes the command and
replaces it with its output. Command substitution is disabled by default, but can be enabled
with the --command-substitution flag.
`,
	Example: `xenv -e NAME=VALUE -e ANOTHER=VALUE2 -E ./.env  --chdir /opt/bin bash -c 'echo $NAME $ANOTHER'`,
	Version: "v0.0.0",
	Run: func(cmd *cobra.Command, args []string) {

		argv := os.Args[1:]

		if len(argv) == 0 {
			for k, v := range env.All() {
				println(k + "=" + v)
			}
			os.Exit(0)
		}

		if len(argv) == 2 && argv[0] == "--split-string" || argv[0] == "-S" {
			argv = cmdargs.Split(argv[1]).ToArray()
		}

		envFlags := pflag.NewFlagSet("", pflag.ContinueOnError)
		envFlags.StringP("split-string", "S", "", "Split a string into key=value pairs, e.g. NAME=VALUE")
		envFlags.StringArrayP("unset", "u", nil, "Unset an environment variable, e.g. NAME")
		envFlags.StringP("chdir", "C", "", "Change the working directory to the specified path")
		envFlags.BoolP("ignore-env", "i", false, "Ignores environment variables.")
		envFlags.StringArrayP("env", "e", nil, "Set an environment variable, e.g. NAME=VALUE")
		envFlags.StringArrayP("env-file", "E", nil, "Load environment variables from a file")
		envFlags.BoolP("command-substitution", "c", false, "Perform command substitution, e.g. `$(command)`")
		envFlags.BoolP("quiet", "q", false, "Suppress error output and only return the exit code or the command output")
		quiet := false
		commandArgs := []string{}
		cmdArgs := []string{}

		envVars := []string{}
		envFiles := []string{}

		inCommand := false
		inArgs := false
		l := len(argv)
		for i := 0; i < len(argv); i++ {
			arg := argv[i]
			if !inArgs && arg[0] != '-' {
				inArgs = true
			}

			if inCommand {
				commandArgs = append(commandArgs, arg)
				continue
			}

			if inArgs {
				if strings.ContainsRune(arg, '=') {
					envVars = append(envVars, arg)
					continue
				}

				base := filepath.Base(arg)
				if strings.HasSuffix(base, ".env") || strings.HasPrefix(base, ".env.") {
					envFiles = append(envFiles, arg)
					continue
				}

				inCommand = true
				commandArgs = append(commandArgs, arg)
				continue
			}

			if arg[0] == '-' {
				if arg == "--quiet" || arg == "-q" {
					quiet = true
					continue
				}

				cmdArgs = append(cmdArgs, arg)
				if arg == "-c" || arg == "--command-substitution" || arg == "-i" || arg == "--ignore-env" {
					continue
				}

				j := i + 1

				if j < l {
					if argv[j][0] != '-' {
						i++
						cmdArgs = append(cmdArgs, argv[i])
						continue
					} else {
						inArgs = true
					}
				}

				continue
			}

			inArgs = true
			if strings.ContainsRune(arg, '=') {
				envVars = append(envVars, arg)
				continue
			}

			base := filepath.Base(arg)
			if strings.HasSuffix(base, ".env") || strings.HasPrefix(base, ".env.") {
				envFiles = append(envFiles, arg)
				continue
			}

			inCommand = true
			commandArgs = append(commandArgs, arg)
		}

		err := envFlags.Parse(cmdArgs)
		if err != nil {
			if !quiet {
				cmd.PrintErrf("Failed to parse flags: %v\n", err)
			}
			os.Exit(1)
		}

		chdir, _ := envFlags.GetString("chdir")
		envVarsValue, _ := envFlags.GetStringArray("env")
		envFilesValue, _ := envFlags.GetStringArray("env-file")
		unsetVars, _ := envFlags.GetStringArray("unset")
		shim, _ := envFlags.GetBool("shim")
		useShell, _ := envFlags.GetString("use-shell")
		commandSubstitution, _ := envFlags.GetBool("command-substitution")
		ignoreEnv, _ := cmd.Flags().GetBool("ignore-env")

		if len(envFilesValue) > 0 {
			envFiles = append(envFiles, envFilesValue...)
		}
		if len(envVarsValue) > 0 {
			envVars = append(envVars, envVarsValue...)
		}

		docs := dotenv.NewDocument()
		for _, envFile := range envFiles {
			absPath, err := resolvePath(envFile)
			if err != nil {
				if !quiet {
					cmd.PrintErrf("Failed to resolve path for env file %s: %v\n", envFile, err)
				}
				os.Exit(1)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				if !quiet {
					cmd.PrintErrf("Env file %s does not exist\n", absPath)
				}
				os.Exit(1)
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				if !quiet {
					cmd.PrintErrf("Failed to read env file %s: %v\n", absPath, err)
				}
				os.Exit(1)
			}

			nextDoc, err := dotenv.Parse(string(content))
			if err != nil {
				if !quiet {
					cmd.PrintErrf("Failed to parse env file %s: %v\n", absPath, err)
				}
				os.Exit(1)
			}

			docs.Merge(nextDoc)
		}

		for _, envVar := range envVars {
			index := strings.Index(envVar, "=")
			if index == -1 {
				if !quiet {
					cmd.PrintErrf("Invalid environment variable format: %s. Expected NAME=VALUE\n", envVar)
				}
				os.Exit(1)
			}

			name := envVar[:index]
			value := envVar[index+1:]
			if name == "" || value == "" {
				if !quiet {
					cmd.PrintErrf("Invalid environment variable format: %s. Expected NAME=VALUE\n", envVar)
				}
				os.Exit(1)
			}

			docs.AddVariable(name, value)
		}

		vars := make(map[string]string)

		if !ignoreEnv {
			if shim {
				shimEnv()
			}

			vars = env.All()

			if len(unsetVars) > 0 {
				for _, unsetVar := range unsetVars {
					delete(vars, unsetVar)
				}
			}
		}

		opts := &env.ExpandOptions{
			Get: func(name string) string {
				if value, ok := vars[name]; ok {
					return value
				}
				return ""
			},
			Set: func(name, value string) error {
				vars[name] = value
				return nil
			},
			CommandSubstitution: commandSubstitution,
		}

		if len(useShell) > 0 {
			opts.UseShell = useShell
			opts.EnableShellExpansion = true
		}

		for _, node := range docs.ToArray() {
			if node.Type == dotenv.VARIABLE_TOKEN {
				namePtr := node.Key
				value := node.Value
				if namePtr == nil {
					if !quiet {
						cmd.PrintErrf("Invalid environment variable name in .env file: %s\n", "")
					}
					os.Exit(1)
				}

				name := *namePtr
				if name == "" {
					if !quiet {
						cmd.PrintErrf("Invalid environment variable name in .env file: %s\n", name)
					}
					os.Exit(1)
				}

				value, err := env.ExpandWithOptions(value, opts)
				if err != nil {
					if !quiet {
						cmd.PrintErrf("Failed to expand environment variable %s: %v\n", *namePtr, err)
					}
					os.Exit(1)
				}

				vars[name] = value
			}
		}

		if len(commandArgs) == 0 {
			if !quiet {
				for k, v := range vars {
					cmd.Println(k + "=" + v)
				}
			}

			os.Exit(0)
		}

		exe := commandArgs[0]
		argz := []string{}
		if len(commandArgs) > 1 {
			argz = commandArgs[1:]
		}

		cmd1 := exec.New(exe, argz...)
		cmd1.Env = []string{}
		cmd1.WithEnvMap(vars)
		if chdir != "" {
			absPath, err := resolvePath(chdir)
			if err != nil {
				if !quiet {
					cmd.PrintErrf("Failed to resolve path for chdir %s: %v\n", chdir, err)
				}
				os.Exit(1)
			}
			cmd1.Dir = absPath
		}

		res, err := cmd1.Run()
		if err != nil {
			if quiet {
				cmd.PrintErrf("Command execution failed: %v\n", err)
			}
			os.Exit(1)
		}

		os.Exit(res.Code)

		// "" "NAME=VALUE"
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

func shimEnv() {
	configDir := env.Get("XDG_CONFIG_HOME")
	dataDir := env.Get("XDG_DATA_HOME")
	cacheDir := env.Get("XDG_CACHE_HOME")

	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		wd, _ := os.Getwd()
		user := os.Getenv("USERNAME")
		hostname := os.Getenv("COMPUTERNAME")
		env.Set("HOME", home)
		env.Set("PWD", wd)
		env.Set("USER", user)
		env.Set("HOSTNAME", hostname)
		if configDir == "" {
			configDir = env.Get("APPDATA")
			env.Set("XDG_CONFIG_HOME", configDir)
		}

		if dataDir == "" {
			dataDir = env.Get("LOCALAPPDATA")
			env.Set("XDG_DATA_HOME", dataDir)
		}

		if cacheDir == "" {
			cacheDir = env.Get("LOCALAPPDATA")
			env.Set("XDG_CACHE_HOME", cacheDir)
		}
	} else {
		if configDir == "" {
			configDir = filepath.Join(env.Get("HOME"), ".config")
			env.Set("XDG_CONFIG_HOME", configDir)
		}

		if dataDir == "" {
			dataDir = filepath.Join(env.Get("HOME"), ".local", "share")
			env.Set("XDG_DATA_HOME", dataDir)
		}

		if cacheDir == "" {
			cacheDir = filepath.Join(env.Get("HOME"), ".cache")
			env.Set("XDG_CACHE_HOME", cacheDir)
		}
	}
}

func resolvePath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	if relativePath[0] == '~' && (relativePath[1] == '/' || relativePath[1] == '\\') {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		relativePath = filepath.Join(homeDir, relativePath[2:])
	}

	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	return filepath.Abs(relativePath)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hx-env.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("split-string", "S", "", "Split a string into command line arguments. Useful for shebangs. Only works with 2 arguments")
	rootCmd.Flags().StringArrayP("unset", "u", nil, "Unset an environment variable, e.g. NAME")
	rootCmd.Flags().StringP("chdir", "C", "", "Change the working directory to the specified path")
	rootCmd.Flags().BoolP("ignore-env", "i", false, "Ignores environment variables.")
	rootCmd.Flags().StringArrayP("env", "e", nil, "Set an environment variable, e.g. NAME=VALUE")
	rootCmd.Flags().StringArrayP("env-file", "E", nil, "Load environment variables from a file")
	rootCmd.Flags().Bool("shim", false, "Shim the environment variables like HOME, USER, PWD, XDG_CONFIG_HOME, etc.")
	rootCmd.Flags().Bool("use-shell", false, "Use shell execution.")
	rootCmd.Flags().BoolP("command-substitution", "c", false, "Perform command substitution, e.g. `$(command)`")
}
