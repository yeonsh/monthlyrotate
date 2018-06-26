# monthlyrotate

Go library for a file whose that rotates monthly. Modified https://github.com/kjk/dailyrotate.

## Usage

```go
pathFormat := filepath.Join("dir", "2006-01.txt")
func onClose(path string, didRotate bool) {
  fmt.Printf("we just closed a file '%s', didRotate: %v\n", path, didRotate)
  if !didRotate {
    return
  }
  // process just closed file e.g. upload to backblaze storage for backup
  go func() {
    // if processing takes a long time, do it in a background goroutine
  }()
}

w, err := monthlyrotate.NewFile(pathFormat, onClose)
panicIfErr(err)

_, err = io.WriteString(w, "hello\n")
panicIfErr(err)

err = w.Close()
panicIfErr(err)
```

Given that files are rotated monthly, you need to provide
a file name format which will be passed to time.Now().UTC().Format()
function. It should produce a unique file name each month e.g. `2006-01.log`

You can also provide `onClose` function which will be called
when the file is closed (either because of rotation or calling Close()).

You can use that to e.g. backup the just rotated file to backblaze.
