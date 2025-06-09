package cluster

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	processor     *Processor
	fsWatcher     *fsnotify.Watcher
	processedFiles sync.Map
	done          chan bool
}

func NewWatcher(processor *Processor) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		processor: processor,
		fsWatcher: fsWatcher,
		done:      make(chan bool),
	}, nil
}

func (w *Watcher) AddDirectory(dir string) error {
	// Add the directory itself
	if err := w.fsWatcher.Add(dir); err != nil {
		return err
	}

	// If recursive, walk and add all subdirectories
	if w.processor.config.Recursive {
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() && path != dir {
				// Check if this directory should be excluded
				if w.processor.shouldExclude(path) {
					return filepath.SkipDir
				}
				
				// Add directory to watcher
				if err := w.fsWatcher.Add(path); err != nil {
					log.Printf("Warning: Could not watch directory %s: %v", path, err)
				}
			}
			return nil
		})
	}

	return nil
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
				if w.isGFSFile(event.Name) && w.matchesPatterns(event.Name) {
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

func (w *Watcher) matchesPatterns(filePath string) bool {
	// Check if file matches any of our node patterns
	for _, pattern := range w.processor.config.NodePatterns {
		// Convert pattern to absolute path for comparison
		// This is a simplified check - in practice we'd need more sophisticated matching
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		
		// Also check if the file path contains pattern elements
		if strings.Contains(filePath, "stats") && strings.Contains(filePath, ".gfs") {
			return true
		}
	}
	return false
}

func (w *Watcher) processFile(filename string) {
	// Check if we've already processed this file recently
	if _, loaded := w.processedFiles.LoadOrStore(filename, true); loaded {
		return
	}

	// Extract node info
	nodeInfo := w.processor.extractNodeInfo(filename)
	
	log.Printf("Processing new cluster GFS file: %s (node=%s, type=%s)", 
		filename, nodeInfo.Name, nodeInfo.Type)
	
	if err := w.processor.processFile(nodeInfo); err != nil {
		log.Printf("Error processing %s: %v", filename, err)
		w.processedFiles.Delete(filename)
	}
}