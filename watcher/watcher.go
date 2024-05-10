package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	notifier       *fsnotify.Watcher
	changeListener func(string, string)
	directories    []string
	patterns       []*regexp.Regexp
}

func NewWatcher(directories []string, extensions []string) *Watcher {
	notifier, _ := fsnotify.NewWatcher()
	patterns := make([]*regexp.Regexp, len(extensions))

	for i, ext := range extensions {
		r := regexp.MustCompile(fmt.Sprintf(`\%s$`, ext))
		patterns[i] = r
	}

	return &Watcher{
		notifier:    notifier,
		directories: directories,
		patterns:    patterns,
	}
}

func (w *Watcher) SetChangeListener(changeListener func(string, string)) *Watcher {
	w.changeListener = changeListener

	return w
}

func (w *Watcher) Start() {
	for _, directory := range w.directories {
		file, err := os.Lstat(directory)
		if err != nil {
			log.Println("ERROR", err)
			continue
		}

		if file.IsDir() {
			// starting at the root of the project, walk each file/directory searching for
			// directories
			if err := filepath.Walk(directory, watchDir(w)); err != nil {
				log.Println("ERROR", err)
				continue
			}
		} else if file.Mode()&os.ModeSymlink == os.ModeSymlink {
			log.Printf("ERROR: %s is a symlink\n", directory)
		} else {
			// TODO: refactor this
			directory, _ := filepath.EvalSymlinks(directory)
			parent := filepath.Dir(directory)
			basename := filepath.Base(directory)
			w.notifier.Add(parent)
			r, err := regexp.Compile(fmt.Sprintf("%s$", regexp.QuoteMeta(basename)))
			if err != nil {
				log.Printf("ERROR: %s is not a valid filename", directory)
				continue
			}
			w.patterns = append(w.patterns, r)
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

				if event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename) {
					if len(w.patterns) == 0 {
						log.Printf("Change detected[%s]: %s\n", event.Op.String(), event.Name)
						w.changeListener(event.Name, event.Op.String())
					} else {
						for _, pattern := range w.patterns {
							if pattern.MatchString(event.Name) {
								log.Printf("Change detected[%s]: %s\n", event.Op.String(), event.Name)
								w.changeListener(event.Name, event.Op.String())
								break
							}
						}
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
				log.Println("ERROR", err)
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
