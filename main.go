package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/gaelph/logstream/runner"
	"github.com/gaelph/logstream/viewport"
	"github.com/gaelph/logstream/watcher"
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

	// creates a new file watcher
	w = watcher.NewWatcher(watchList, extensionList)
	defer w.Close()

	p = viewport.NewProgram()
	r := runner.NewRunner(command, delay)
	r.SetDelegate(p)

	onChange := func(path string, change string) {
		p.Append(fmt.Sprintf("Change detected[%s]: %s\n", change, path))
		r.Restart()
	}
	w.SetChangeListener(onChange)

	w.Start()
	time.Sleep(time.Duration(3000) * time.Millisecond)

	go r.Start()
	p.Run()

	r.Stop()
}
