package watcher

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	notifier       *fsnotify.Watcher
	changeListener func(string, string)
	directories    []string
	extensions     []string
}

func NewWatcher(directories []string, extensions []string) *Watcher {
	notifier, _ := fsnotify.NewWatcher()
	return &Watcher{
		notifier:    notifier,
		directories: directories,
		extensions:  extensions,
	}
}

func (w *Watcher) SetChangeListener(changeListener func(string, string)) *Watcher {
	w.changeListener = changeListener

	return w
}

func (w *Watcher) Start() {
	for _, directory := range w.directories {
		// starting at the root of the project, walk each file/directory searching for
		// directories
		if err := filepath.Walk(directory, watchDir(w)); err != nil {
			fmt.Println("ERROR", err)
		}
	}

	go func() {
		for {
			select {
			// watch for events
			case event := <-w.notifier.Events:
				if event.Op.Has(fsnotify.Remove) || event.Op.Has(fsnotify.Rename) {
					w.notifier.Remove(event.Name)
				}

				fileExt := filepath.Ext(event.Name)
				for _, ext := range w.extensions {
					if ext == fileExt && (event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename)) {
						w.changeListener(event.Name, event.Op.String())
						break
					}
				}

				fileInfo, err := os.Stat(event.Name)
				if err != nil {
					continue
				}

				if fileInfo.IsDir() {
					if event.Op.Has(fsnotify.Create) {
						w.notifier.Add(event.Name)
					}
					if event.Op.Has(fsnotify.Remove) {
						w.notifier.Remove(event.Name)
					}
				}

				// watch for errors
			case err := <-w.notifier.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}()
}

func (w *Watcher) Close() {
	w.notifier.Close()
}

// watchDir gets run as a walk func, searching for directories to add watchers to
func watchDir(w *Watcher) func(string, os.FileInfo, error) error {
	return func(path string, fi os.FileInfo, err error) error {

		// since fsnotify can watch all the files in a directory, watchers only need
		// to be added to each nested directory
		if fi.Mode().IsDir() {
			return w.notifier.Add(path)
		}

		return nil
	}
}
