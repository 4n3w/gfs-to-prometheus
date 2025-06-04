package watcher

import (
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/4n3w/gfs-to-prometheus/internal/converter"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	converter      *converter.Converter
	fsWatcher      *fsnotify.Watcher
	processedFiles sync.Map
	done           chan bool
}

func New(conv *converter.Converter) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		converter: conv,
		fsWatcher: fsWatcher,
		done:      make(chan bool),
	}, nil
}

func (w *Watcher) AddDirectory(dir string) error {
	return w.fsWatcher.Add(dir)
}

func (w *Watcher) Start() error {
	go w.watch()
	<-w.done
	return nil
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.fsWatcher.Close()
}

func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if w.isGFSFile(event.Name) {
					log.Printf("Detected GFS file: %s", event.Name)
					go w.processFile(event.Name)
				}
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)

		case <-w.done:
			return
		}
	}
}

func (w *Watcher) isGFSFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".gfs"
}

func (w *Watcher) processFile(filename string) {
	if _, loaded := w.processedFiles.LoadOrStore(filename, true); loaded {
		return
	}

	log.Printf("Processing new GFS file: %s", filename)
	if err := w.converter.ConvertFile(filename); err != nil {
		log.Printf("Error processing %s: %v", filename, err)
		w.processedFiles.Delete(filename)
	}
}