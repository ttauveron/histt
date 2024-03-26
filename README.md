# histt

This project is a basic reimplementation of https://github.com/dvorka/hstr in Go.
It provides an easy and flexible way to work with command line history. With support for customizable history file locations and enhanced search capabilities, it streamlines the process of finding and managing past commands.

![image](https://github.com/ttauveron/histt/assets/1558361/7fcbcfcc-2c52-4d6a-b077-894cfd50f952)

## Features

- *Customizable History File Location*: Set the history file location using the HISTORY_LOCATION environment variable, or rely on the default location at ~/.bash_history.
- *Enhanced Search Capabilities*: Search your command history using exact matches, keywords, or regular expressions to quickly find what you need.
- *Terminal UI*: A user-friendly terminal interface makes navigating and selecting command history straightforward.

## Installation

To get started, browse https://github.com/ttauveron/histt/releases and download the executable for your system.

Alternatively clone this repository and build the tool using Go:
```
git clone https://github.com/ttauveron/histt.git
cd histt
go build
```

### Cross-Compilation

```
GOOS=linux GOARCH=amd64 go build -o histt-linux-amd64 main.go
GOOS=darwin GOARCH=arm64 go build -o histt-macos-arm64 main.go
```

## Configuration

Optionally, you can configure the location of the history file by setting the `HISTORY_LOCATION` environment variable:
```
export HISTORY_LOCATION="/path/to/your/history/file"
```

If `HISTORY_LOCATION` is not set, the tool defaults to using `~/.bash_history`.

Add the following configuration to your `.bashrc`:

```
# Append to the history file, don't overwrite it
shopt -s histappend

# Save multi-line commands as one command
shopt -s cmdhist

# Huge history. Doesn't appear to slow things down, so why not?
HISTSIZE=500000
HISTFILESIZE=10000000

# Avoid duplicate entries
HISTCONTROL="erasedups:ignoreboth"

# Ensure synchronization between Bash memory and history file
export PROMPT_COMMAND="history -a; history -n; ${PROMPT_COMMAND}"
# if this is interactive shell, then bind histt to Ctrl-r
if [[ $- =~ .*i.* ]]; then bind '"\C-r": "\C-a histt -- \C-j"'; fi
```

## Keyboard Shortcuts

- *Ctrl+E*: Switch between exact matching, keyword search, and regex search modes.
- *Ctrl+T*: Toggle case sensitivity for searches.
