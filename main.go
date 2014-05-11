package globwatch

import (
	"os"
	"path/filepath"
	"time"
)

type EvType int

const (
	ADDED EvType = iota
	DELETED
	TRUNCATED
)

// Event type sent on the channel returned by Watch()
type Event struct {
	typ      EvType
	filename string
}

// Get the type of an event
func (e Event) Type() EvType {
	return e.typ
}

// Get the filename the event relates to
func (e Event) Filename() string {
	return e.filename
}

// Used to track which files are being watched
// and track changes in filesize.
type file struct {
	prev os.FileInfo
	curr os.FileInfo
}

// Watch a glob pattern in a go routine,
// emitting changes as events on the returned channel
func Watch(pattern string, sleepInMs int) <-chan Event {
	out := make(chan Event)
	watchedFiles := make(map[string]*file)
	sleepTime := time.Duration(sleepInMs) * time.Millisecond

	go func() {
		for {
			currentFiles, err := filepath.Glob(pattern)
			if err != nil {
				time.Sleep(sleepTime)
				continue
			}

			// Check the watched files for deletions and truncations
			for filename, file := range watchedFiles {
				fi, err := getFileInfo(filename)
				if err != nil {
					// It's been deleted
					delete(watchedFiles, filename)
					out <- Event{DELETED, filename}
					continue
				}

				// Add the current file info
				file.prev = file.curr
				file.curr = fi

				if file.prev != nil && file.curr.Size() < file.prev.Size() {
					// It's been truncated
					out <- Event{TRUNCATED, filename}
				}
			}

			// Check for new files
			for _, candidate := range currentFiles {
				_, exists := watchedFiles[candidate]
				if exists {
					continue
				}
				fi, err := getFileInfo(candidate)
				if err != nil {
					continue
				}
				watchedFiles[candidate] = &file{fi, fi}
				out <- Event{ADDED, candidate}
			}

			// Sleep a bit to not hammer the disk quite so much
			time.Sleep(sleepTime)
		}
	}()

	return out
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
