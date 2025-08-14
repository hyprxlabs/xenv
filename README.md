# xenv

A cross-platform cli-app for setting environment variables and executing commands with
the newly set environment variables.

The `xenv` command allows you to set environment variables from the command line,
load them from `.env` files, or both, and execute a command with those environment variables.

The environment variables support environment variable expansion and command substitution.

## Installation

```bash
go install github.com/hyprxlabs/xenv@latest
```

## Usage

```bash
xenv [OPTIONS] [COMMAND] [ARG...]
```

## Options

- `--chdir`, `-C`: Change the working directory to the specified path.
- `--ignore-env`, `-i`: Removes all environment variables.
- `--env`, `-e`: Set an environment variable, e.g. `NAME=VALUE`.
- `--env-file`, `-E`: Load environment variables from a file.
- `--unset`, `-u`: Unset an environment variable, e.g. `NAME`.
- `--command-substitution`, `-c`: Perform command substitution, e.g. `'VAR=$(command)'`.
- `--split-string`, `-S`: Split a string into key=value pairs, e.g. `NAME=VALUE`.

## Variable Expansion

- `$VAR`: Expands to the value of the environment variable `VAR`.
- `${VAR}`: Same as above, but allows for more complex expressions.
- `${VAR:-default}`: If `VAR` is unset or null, expands to `default`.
- `${VAR:=default}`: If `VAR` is unset or null, sets `VAR` to `default` and expands to `default`.
- `${VAR:?error}`: If `VAR` is unset or null, prints `error` and exits.

## Command Substitution

- $(command): Executes `command` and substitutes its output. Currently only supports single-line commands
  and no variable expansion within the command.

## Examples

```bash
xenv -c -e NAME=VALUE -e 'VAR=$(command)' -S 'KEY=VAL' bash -c 'echo $NAME $VAR $KEY'
```

The `--env`, `-e`, --env-file, `-E`, are not explicity required as the commandline tool will
inspect the argument and if the argument contains an `=` or a `.env` in the filename, it will
automatically set the environment variable or load the `.env` file.

These options exist to enable explicit declaring intent and work around potential edge cases.

```bash
xenv -c NAME=VALUE 'VAR=$(command)' bash -c 'echo $NAME $VAR $KEY'
```

Unsetting an environment variable:

```bash
xenv -u PWD bash -c 'echo $PWD'
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.
