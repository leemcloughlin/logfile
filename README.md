
# logfile
    import "github.com/leemcloughlin/logfile"

LogFile is an output handler for the standard Go (golang) log library to allow logging
to a file.

LogFile supports the following features:


	Writes to the log file are buffered but are automatically flushed.
	
	Safe to use with multiple goroutines.
	
	Log files can have a maxium size and on reaching it the file can either be
	truncated or "rotated" (moved aside: log -> log.1, log.1 -> log.2...).
	
	A log file can be moved aside outside of the program (perhaps by something
	like Linux's logrotate) and LogFile will detect this and close/reopen the file
	
	A default rotate function is provided but users can provide their own (see
	RotateFile). The default uses the OldVersions value to decide how many
	versions to keep. Note that writing to the log is blocked while rotating so
	keep RotateFile quick.
	
	By default messages are still sent to standard error as well as the file
	
	There are command line flags to override default behavior (requires
	flag.Parse to be called)
	
	Actually buffering can result in a lot less writes which is useful on devices
	(like flash memory) that have limited write cycles. The downside is that
	messages may be lost on panic or unplanned exit.

Note that LogFile creates a goroutine on New. To ensure its deleted call Close

Command line arguments:


	  -logcheckseconds int
	    	Default seconds to check log file still exists (default 60)
	  -logfile string
	    	Use as the filename for the first LogFile created without a filename
	  -logflushseconds int
	    	Default seconds to wait before flushing pending writes to the log file (default -1)
			If <= 0 then the log is writen before returning.
	  -logmax int
	    	Default maximum file size, 0 = no limit
	  -lognostderr
	    	Default to no logging to stderr
	  -logversions int
	    	Default old versions of file to keep (otherwise deleted)

Example:


	// was -logfile passed?
	if logfile.Defaults.FileName != "" {
		logFileName = logfile.Defaults.FileName
	}
	
	logFile, err := logfile.New(
		&logfile.LogFile{
			FileName: logFileName,
			MaxSize:  500 * 1024, // 500K duh!
			Flags:    logfile.FileOnly | logfile.OverWriteOnStart})
	if err != nil {
		log.Fatalf("Failed to create logFile %s: %s\n", logFileName, err)
	}
	
	log.SetOutput(logFile)
	
	log.Print("hello")
	logFile.Close()




## Constants
``` go
const (
    // Flags
    FileOnly         = 1 << iota // Log only to file, not to stderr
    OverWriteOnStart             // Note the default is to append
    RotateOnStart
    NoErrors // Disables printing internal errors to stderr

)
```

## Variables
``` go
var (
    // These are the defaults for a LogFile. Most can be overridden on the command line
    Defaults = LogFile{
        Flags:        0,
        FileName:     "",
        FileMode:     0644,
        MaxSize:      0,
        OldVersions:  0,
        CheckSeconds: 60,
        FlushSeconds: -1,
    }
    NoStderr = false
)
```

## func FileNameVersion
``` go
func FileNameVersion(fileName string, v int) string
```
FileNameVersion returns a versioned log file name for rotating.
If v is zero it returns the file name unmodified.
Otherwise it add .v to the end



## type LogFile
``` go
type LogFile struct {
    // Flags override default behaviour (see also command line flag -lognostderr)
    Flags int

    // FileName to write to.
    // See also the -logfile command line flag
    FileName string

    // FileMode for any newly created log files
    FileMode os.FileMode

    // If MaxSize is non zero and if log file is about to become bigger than
    // MaxSize it will be closed, passed to RotateFile, then a new, empty
    // log file will be created and opened.
    // See also the -logmax command line flag
    MaxSize int64

    // CheckSeconds is how often LogFile will test to see if the log file
    // still exists as it may have been moved aside by something like Linux's
    // logrotate.
    // Note that a checking file existance is little expensive on a most
    // Linux systems so limiting checking is a good option.
    // On calling New if this is zero the default value (60) will be used
    // See also the -logcheckseconds command line flag
    CheckSeconds int

    // RotateFileFunc is called whenever the log file needs "rotating"
    // (moved aside: log -> log.1, log.1 -> log.2...)
    // Rotating could be because the file is about to exceed MaxSize or
    // because logging is just starting and the RotateOnStart flag is set.
    // If nil a default is provided that rotates up to a OldVerions and deletes
    // any older.
    // Never call this directly. If you need to rotate logs call lp.RotateFile()
    RotateFileFunc func()

    // When the default RotateFile is called this is the number of old versions
    // to keep.
    // See also the -logversions command line flag
    OldVersions int

    // FlushSeconds is how often the log file is writen out. Note that the log
    // file will be writen to immdiately if the buffer gets full or on the log
    // file being closed.
    // If FlushSeconds is zero the default value is used. If less than zero
    // the log file will be flushed after every write
    // CAUTION: If not the default (-1) then writes are buffered and may not be
    // writen out if the program exits/panics
    FlushSeconds int
    // contains filtered or unexported fields
}
```
LogFile implements an io.Writer so can used by the standard log library









### func New
``` go
func New(lp *LogFile) (*LogFile, error)
```
New creates, if necessary, and opens a log file.
If a LogFile is passed any empty fields are filled with suitable defaults.
If nil is passed an empty LogFile is created and then filled in.
Once finished with the LogFile call Close()




### func (\*LogFile) Close
``` go
func (lp *LogFile) Close()
```
Close flushs any pending data out and then closes a log file opened by calling New()



### func (\*LogFile) Flush
``` go
func (lp *LogFile) Flush()
```
Flush writes any pending log entries out



### func (\*LogFile) PrintError
``` go
func (lp *LogFile) PrintError(format string, args ...interface{})
```
PrintError prints out internal errors to standard error (if not turned off by the NoErrors flag)



### func (\*LogFile) RotateFile
``` go
func (lp *LogFile) RotateFile()
```
RotateFile requests an immediate file rotation.



### func (\*LogFile) RotateFileFuncDefault
``` go
func (lp *LogFile) RotateFileFuncDefault()
```
RotateFileFuncDefault only rotates if OldVersions non zero.
It deletes the oldest version and renames the others log -> log.1, log.1 -> log.2...



### func (\*LogFile) Write
``` go
func (lp *LogFile) Write(p []byte) (n int, err error)
```
Write is called by Log to write log entries.









- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)