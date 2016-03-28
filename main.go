package globwatch

import (
	"os"
	"path/filepath"
	"time"
)

// EvType represents the type of event, e.g. Added, Deleted
type EvType int

const (
	// Added signifies a file was added
	Added EvType = iota
	// Deleted signifies a file was deleted
	Deleted
	// Truncated signifies a file was truncated
	Truncated
)

// Event type sent on the channel returned by Watch()
type Event struct {
	typ      EvType
	filename string
}

// Type return the type of an event
func (e Event) Type() EvType {
	return e.typ
}

// Filename returns the filename an event related to
func (e Event) Filename() string {
	return e.filename
}

// Used to track which files are being watched
// and track changes in filesize.
type file struct {
	prev os.FileInfo
	curr os.FileInfo
}

type fileMap map[string]*file

func (m fileMap) add(fn string) {
	fi, err := getFileInfo(fn)
	if err != nil {
		return
	}
	m[fn] = &file{fi, fi}
}

func (m fileMap) remove(fn string) {
	delete(m, fn)
}

func (m fileMap) exists(fn string) bool {
	_, exists := m[fn]
	return exists
}

// Watch a glob pattern in a goroutine
// Returns a channel of events and a control channel to stop the watching
func Watch(pattern string, sleepInMs int) (<-chan Event, chan<- bool) {
	out := make(chan Event)
	stop := make(chan bool)
	watches := make(fileMap)
	sleepTime := time.Duration(sleepInMs) * time.Millisecond

	// Fn wrapping the select over emitting/stopping
	emit := func(typ EvType, fn string) bool {
		select {
		case out <- Event{typ, fn}:
			return true
		case <-stop:
			return false
		}
	}

	// Fn wrapping the selection over 'wait or stop'
	wait := func(d time.Duration) bool {
		select {
		case <-time.After(d):
			return true
		case <-stop:
			return false
		}
	}

	// Let's go and watch that glob!
	go func() {
		defer close(out)

		for {
			currentFiles, err := filepath.Glob(pattern)
			if err != nil {
				// Wait to retry or stop
				if wait(sleepTime) {
					continue
				} else {
					return
				}
			}

			// Check the watched files for deletions and truncations
			for filename, file := range watches {
				fi, err := getFileInfo(filename)
				if err != nil {
					// It's been deleted
					watches.remove(filename)
					if !emit(Deleted, filename) {
						return
					}
					continue
				}

				// Add the current file info
				file.prev = file.curr
				file.curr = fi

				if file.prev != nil && file.curr.Size() < file.prev.Size() {
					// It's been truncated
					if !emit(Truncated, filename) {
						return
					}
				}
			}

			// Check for new files
			for _, candidate := range currentFiles {
				if watches.exists(candidate) {
					continue
				}

				watches.add(candidate)

				if !emit(Added, candidate) {
					return
				}
			}

			// Sleep a bit to not hammer the disk quite so much
			if !wait(sleepTime) {
				return
			}
		}
	}()

	return out, stop
}

// Get the os.FileInfo struct returned by (*os.File).Stat()
func getFileInfo(filename string) (os.FileInfo, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	s, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	return s, nil
}
