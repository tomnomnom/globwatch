package globwatch

import (
	"os"
	"path/filepath"
	"time"
)

const (
	glob_retry_sleep = 1000 * time.Millisecond
)

type EvType int

const (
	ADDED EvType = iota
	DELETED
	TRUNCATED
)

type Event struct {
	typ      EvType
	filename string
}

func (e Event) Type() EvType {
	return e.typ
}

func (e Event) Filename() string {
	return e.filename
}

type file struct {
	prev os.FileInfo
	curr os.FileInfo
}

func Watch(pattern string) <-chan Event {
	out := make(chan Event)
	watchedFiles := make(map[string]*file)

	go func() {
		for {
			currentFiles, err := filepath.Glob(pattern)
			if err != nil {
				time.Sleep(glob_retry_sleep)
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
			time.Sleep(glob_retry_sleep)
		}
	}()

	return out
}

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
