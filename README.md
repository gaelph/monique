# Monique

Watch any thing, execute any thing, filter and search through the output. Live.

> This is early days for `monique`. Things might break, things are not documented.

## Why?

Sometimes, you have a program you relaunch on file changes, with `nodemon`,
`watchexec`,  `entr` or whatever. And sometimes that program produces lots of output,
but your only want to watch the lines that match a certain pattern.
You can use `grep`, `awk` but if the pattern you are looking for changes,
you have to re-run the command.

With `monique`, you can watch for file changes, execute your program,
and filter the output, live. Change the pattern, and the output will be updated.
You can even search the filtered output.

## Usage

```sh
# start watching for file changes and run <command>
monique [-watch <file-or-directory> [-exts <list-of-extensions>]] <command>
```

### Options

`-watch <file-or-directory>`: A path to file or directory to watch.
There can be multiple `-watch` arguments if you want to watch multiple things.
There can be none. In that case, `<command>` will be executed once

`-exts <list-of-extensions>`: A comma separated list of extensions to watch, like
`.go,.js,.py` to watch for go, javascript and python files.
It is ignored if `-watch` is absent.

`<command>`: The command to execute

### Examples

```sh
# run make when a .swift or .py file changes
monique -watch ./Sources -exts .swift,.py make
```

```sh
# filter and search live in the output of a tail -f call
monique tail -f
```

### Key bindings
While watching the output, there are several things you can do:

- `Ctrl-C`: Quit
- `Ctrl-R`: Restart the command
- `Ctrl-D`: Scroll Down
- `Ctrl-U`: Scroll Up

While the input field is not focused, you can use the following keys:

- `f`: Start filtering
- `/`: Start searching
- `n`: Jump to the next search match (from bottom to top)
- `N`: Jump to the previous search match

While the input field is focused, you can use the following keys:

- `Esc`: Clear the filter or search input and loose focus
- `Enter`: Keep the filter or search input and loose focus

### Filtering and Searching pattern
Currently, it uses the default golang regexp package to parse the filter and
search patterns.

Monique uses "smart sensitivity" when it comes to case. It means, that it will
be case sensitive if your filter or search pattern contains at least one
capital letter, but is case insensitive otherwise.

Monique also adds a "top-level capture group", which let's you type patterns
like: `DEBUG|TRACE` without parenthesis, if all you are looking for are lines
containing either of those to words

## Acknowledgments

Uses:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [PTY](https://github.com/creack/pty)
