package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gaelph/monique/mediator"
	"github.com/gaelph/monique/runner"
	"github.com/gaelph/monique/viewport"
	"github.com/gaelph/monique/watcher"
)

type watchTargets []string

func (w *watchTargets) String() string {
	return strings.Join(*w, "\n")
}
func (w *watchTargets) Set(value string) error {
	*w = append(*w, value)
	return nil
}

var p *viewport.Program

var w *watcher.Watcher

func main() {
	var watchList watchTargets
	var delay int
	var exts string
	var extensionList []string
	var command []string
	var showHelp bool

	flag.Var(&watchList, "watch", "path to a directory to watch")
	flag.Var(&watchList, "w", "shorthand for -watch")
	flag.StringVar(&exts, "exts", "", "file extensions")
	flag.StringVar(&exts, "e", "", "shorthand for -exts")
	flag.IntVar(&delay, "delay", 100, "delay in ms")
	flag.IntVar(&delay, "d", 100, "shorthand for -delay")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.BoolVar(&showHelp, "h", false, "shorthand for -help")

	flag.Parse()
	command = flag.Args()

	if showHelp {
		printHelp()
		os.Exit(0)
		return
	}

	extensionList = strings.Split(exts, ",")
	for idx, ext := range extensionList {
		extensionList[idx] = strings.TrimSpace(ext)
	}

	m := mediator.NewMediator()

	p = viewport.NewProgram(strings.Join(command, " "), m)
	r := runner.NewRunner(command, delay)
	r.SetMediator(m)

	if len(watchList) > 0 {
		// creates a new file watcher
		w = watcher.NewWatcher(watchList, extensionList)
		defer w.Close()

		onChange := func(path string, change string) {
			p.Append(fmt.Sprintf("Change detected[%s]: %s\n", change, path))
			m.SendRequestRestart()
		}
		w.SetChangeListener(onChange)

		w.Start()
	}

	go r.Start()
	p.Run()

	r.Stop()
}

func printHelp() {
	fmt.Fprint(os.Stderr, `monique - execute commands when files change, filter the output, live

Usage:  monique [options] <command>
  monique <command>
  monique [[-watch <path>]... [-exts <ext-list>] [-delay <delay>]  <command>

Examples:
  - Restart a command when any js or css file changes in a single directory:
    $ monique -watch ./src -exts .js,.css npm run start

  - Run 'make' when any c, cpp, or header file changes in two directories:
    $ monique -watch ./src -watch ./include -exts .c,.cpp,.h,.hpp make

  - Filter and search on a tail -f call, live:
    $ monique tail -f /var/log/nginx/access.log

Options:
`)
	flag.PrintDefaults()
}
