package globwatch

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestHappy(t *testing.T) {

	// Create a dir to put the test files in
	tmpdir, err := ioutil.TempDir("", "globwatch-")
	if err != nil {
		t.Errorf("Failed to create tmpdir:", err)
	}
	defer os.RemoveAll(tmpdir)

	// Add an initial file
	one := writeTestFile(tmpdir, "one.log", "File one")

	// Watch the tmpdir for *.log files
	evs := Watch(tmpdir + "/*.log", 0)
	ev := <-evs

	// Check the file event
	if ev.Type() != ADDED {
		t.Errorf("Should have received ADDED event")
	}
	if ev.Filename() != one {
		t.Errorf("Event filename should have been %s, was %s", one, ev.Filename())
	}

	// Add a second file
	two := writeTestFile(tmpdir, "two.log", "File two")
	ev = <-evs
	if ev.Type() != ADDED {
		t.Errorf("Should have received ADDED event")
	}
	if ev.Filename() != two {
		t.Errorf("Event filename should have been %s, was %s", two, ev.Filename())
	}

	// Truncate the second file
	os.Truncate(two, 0)
	ev = <-evs
	if ev.Type() != TRUNCATED {
		t.Errorf("Should have received TRUNCATED event")
	}

	// Remove the first file
	os.Remove(one)
	ev = <-evs
	if ev.Type() != DELETED {
		t.Errorf("Should have received DELETED event")
	}
}

func writeTestFile(dir string, name string, content string) string {
	path := dir+"/"+name
	ioutil.WriteFile(path, []byte(name), os.FileMode(0777))
	return path
}
