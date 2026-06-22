package cmd

import (
	"crypto/sha256"
	"io"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const ignorePattern = `\.swp$|~$|^\.DS_Store$|^4913$`
const lockTime = 3 * time.Second

var (
	hashMu    sync.Mutex
	hashCache = make(map[string][sha256.Size]byte)
)

func fileHash(path string) ([sha256.Size]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [sha256.Size]byte{}, err
	}

	var sum [sha256.Size]byte
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func contentChanged(path string) bool {
	newHash, err := fileHash(path)
	if err != nil {
		return true
	}

	hashMu.Lock()
	defer hashMu.Unlock()

	prev, exists := hashCache[path]
	hashCache[path] = newHash
	return !exists || prev != newHash
}

func createWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return watcher, err
	}
	logInfo("Watching %s/ for changes", dir)
	err = watcher.Add(dir)
	return watcher, err
}

func watch(done <-chan interface{}, errorChan chan<- error, reload chan<- bool, watcher *fsnotify.Watcher) {
	isLocked := false
	for {
		select {
		case event := <-watcher.Events:
			if isLocked {
				break
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				r := regexp.MustCompile(ignorePattern)
				if r.MatchString(event.Name) {
					logDebug("Debug [ignore]: `%s`", event.Name)
				} else if !contentChanged(event.Name) {
					logDebug("Debug [unchanged]: `%s`", event.Name)
				} else {
					logInfo("Change detected in %s, refreshing", event.Name)
					isLocked = true
					reload <- true
					timer := time.NewTimer(lockTime)
					go func() {
						<-timer.C
						isLocked = false
					}()
				}
			}
		case err := <-watcher.Errors:
			errorChan <- err
		case <-done:
			return
		}
	}
}
