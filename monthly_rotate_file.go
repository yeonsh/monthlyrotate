// Package monthlyrotate provides a file that is rotated monthly (at midnight UTC).
//
// You provide a pattern for a file path. That pattern will be formatted with
// time.Format to generate a real path. It should be unique for each month e.g.
// 2020-01.txt.
//
// You Write to a file and the code takes care of closing existing file and
// opening a new file when we're crossing monthly boundaries.
package monthlyrotate

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File describes a file that gets rotated monthly
type File struct {
	sync.Mutex
	pathFormat string

	// info about currently opened file
	year    int
	month   int
	path    string
	file    *os.File
	onClose func(path string, didRotate bool)

	// position in the file of last Write or Write2, exposed for tests
	lastWritePos int64
}

func (f *File) close(didRotate bool) error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	if err == nil && f.onClose != nil {
		f.onClose(f.path, didRotate)
	}
	f.year = 0
	f.month = 0
	return err
}

func (f *File) open() error {
	t := time.Now().UTC()
	f.path = t.Format(f.pathFormat)
	f.year = t.Year()
	f.month = int(t.Month())

	// we can't assume that the dir for the file already exists
	dir := filepath.Dir(f.path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// would be easier to open with os.O_APPEND but Seek() doesn't work in that case
	flag := os.O_CREATE | os.O_WRONLY
	f.file, err = os.OpenFile(f.path, flag, 0644)
	if err != nil {
		return err
	}
	_, err = f.file.Seek(0, io.SeekEnd)
	return err
}

// rotate on new day
func (f *File) reopenIfNeeded() error {
	t := time.Now().UTC()
	if t.Year() == f.year && int(t.Month()) == f.month {
		return nil
	}
	err := f.close(true)
	if err != nil {
		return err
	}
	return f.open()
}

// NewFile creates a new file that will be rotated daily (at UTC midnight).
// pathFormat is file format accepted by time.Format that will be used to generate
// a name of the file. It should be unique in a given day e.g. 2006-01-02.txt.
// onClose is an optional function that will be called every time existing file
// is closed, either as a result calling Close or due to being rotated.
// didRotate will be true if it was closed due to rotation.
// If onClose() takes a long time, you should do it in a background goroutine
// (it blocks all other operations, including writes)
func NewFile(pathFormat string, onClose func(path string, didRotate bool)) (*File, error) {
	f := &File{
		pathFormat: pathFormat,
	}
	// force early failure if we can't open the file
	// note that we don't set onClose yet so that it won't get called due to
	// opening/closing the file
	err := f.reopenIfNeeded()
	if err != nil {
		return nil, err
	}
	err = f.close(false)
	if err != nil {
		return nil, err
	}
	f.onClose = onClose
	return f, nil
}

// Close closes the file
func (f *File) Close() error {
	f.Lock()
	defer f.Unlock()
	return f.close(false)
}

func (f *File) write(d []byte, flush bool) (int64, int, error) {
	err := f.reopenIfNeeded()
	if err != nil {
		return 0, 0, err
	}
	f.lastWritePos, err = f.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, 0, err
	}
	n, err := f.file.Write(d)
	if err != nil {
		return 0, n, err
	}
	if flush {
		err = f.file.Sync()
	}
	return f.lastWritePos, n, err
}

// Write writes data to a file
func (f *File) Write(d []byte) (int, error) {
	f.Lock()
	defer f.Unlock()
	_, n, err := f.write(d, false)
	return n, err
}

// Write2 writes data to a file, optionally flushes. To enable users to later
// seek to where the data was written, it returns name of the file where data
// was written, offset at which the data was written, number of bytes and error
func (f *File) Write2(d []byte, flush bool) (string, int64, int, error) {
	f.Lock()
	defer f.Unlock()
	writtenAtPos, n, err := f.write(d, flush)
	return f.path, writtenAtPos, n, err
}

// Flush flushes the file
func (f *File) Flush() error {
	f.Lock()
	defer f.Unlock()
	return f.file.Sync()
}
