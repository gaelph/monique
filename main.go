package main

import (
	"flag"
	"fmt"
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

var watchList watchTargets
var dir string
var delay int
var exts string
var extensionList []string
var command []string

var p *viewport.Program

var w *watcher.Watcher

func main() {
	flag.Var(&watchList, "watch", "path to a directory to watch")
	flag.StringVar(&exts, "exts", "", "file extensions")
	flag.IntVar(&delay, "delay", 100, "delay in ms")

	flag.Parse()
	command = flag.Args()

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
