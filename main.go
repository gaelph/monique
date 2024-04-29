package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/gaelph/logstream/runner"
	"github.com/gaelph/logstream/viewport"
	"github.com/gaelph/logstream/watcher"
)

var dir string
var delay int
var exts string
var extensionList []string
var command []string

var p *viewport.Program

var w *watcher.Watcher

func main() {
	flag.StringVar(&dir, "dir", "", "path to the directory to watch")
	flag.StringVar(&exts, "exts", "", "file extensions")
	flag.IntVar(&delay, "delay", 100, "delay in ms")

	flag.Parse()
	command = flag.Args()

	extensionList = strings.Split(exts, ",")
	for idx, ext := range extensionList {
		extensionList[idx] = strings.TrimSpace(ext)
	}

	dirs := make([]string, 1)
	dirs[0] = dir

	// creates a new file watcher
	w = watcher.NewWatcher(dirs, extensionList)
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

	go r.Start()
	p.Run()

	r.Stop()
}
