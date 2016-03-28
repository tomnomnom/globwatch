package globwatch

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestHappy(t *testing.T) {

	// Create a dir to put the test files in
	tmpdir := createTestDir(t)
	defer os.RemoveAll(tmpdir)

	// Add an initial file
	one := writeTestFile(tmpdir, "one.log", "File one")

	// Watch the tmpdir for *.log files
	evs, _ := Watch(tmpdir+"/*.log", 1)
	ev := <-evs

	// Check the file event
	if ev.Type() != Added {
		t.Errorf("Should have received ADDED event")
	}
	if ev.Filename() != one {
		t.Errorf("Event filename should have been %s, was %s", one, ev.Filename())
	}

	// Add a second file
	two := writeTestFile(tmpdir, "two.log", "File two")
	ev = <-evs
	if ev.Type() != Added {
		t.Errorf("Should have received ADDED event")
	}
	if ev.Filename() != two {
		t.Errorf("Event filename should have been %s, was %s", two, ev.Filename())
	}

	// Truncate the second file
	err := os.Truncate(two, 0)
	if err != nil {
		t.Errorf("Failed to truncate second file")
	}
	ev = <-evs
	if ev.Type() != Truncated {
		t.Errorf("Should have received TRUNCATED event")
	}

	// Remove the first file
	err = os.Remove(one)
	if err != nil {
		t.Errorf("Failed to remove first file")
	}
	ev = <-evs
	if ev.Type() != Deleted {
		t.Errorf("Should have received DELETED event")
	}
}

func TestStop(t *testing.T) {
	tmpdir := createTestDir(t)
	defer os.RemoveAll(tmpdir)

	evs, stop := Watch(tmpdir+"/*.log", 0)

	stop <- true
	_, stillOpen := <-evs
	if stillOpen {
		t.Errorf("Sending stop signal should close the events channel")
	}
}

func createTestDir(t *testing.T) string {
	tmpdir, err := ioutil.TempDir("", "globwatch-")
	if err != nil {
		t.Errorf("Failed to create tmpdir: %s", err)
	}
	return tmpdir
}

func writeTestFile(dir string, name string, content string) string {
	path := dir + "/" + name
	_ = ioutil.WriteFile(path, []byte(name), os.FileMode(0777))
	return path
}
