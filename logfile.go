/*
File summary: logfile interface
Package: logfile
Author: Lee McLoughlin

Copyright (C) 2015 LMMR Tech Ltd All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
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
*/
package logfile

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	// These are the defaults for a LogFile. Most can be overridden on the command line
	Defaults = LogFile{
		Flags:        0,
		FileName:     "",
		FileMode:     0644,
		MaxSize:      0,
		OldVersions:  0,
		CheckSeconds: 60,
		FlushSeconds: -1, // Immediate write is the default
	}
	NoStderr = false

	errorSeconds        = 60
	defaultFileNameUsed = false
)

const (
	// Flags
	FileOnly         = 1 << iota // Log only to file, not to stderr
	OverWriteOnStart             // Note the default is to append
	RotateOnStart
	NoErrors // Disables printing internal errors to stderr

	truncateLog   = true
	noTruncateLog = false
)

func init() {
	flag.StringVar(&Defaults.FileName, "logfile", Defaults.FileName, "Use as the filename for the first LogFile created without a filename")
	flag.Int64Var(&Defaults.MaxSize, "logmax", Defaults.MaxSize, "Default maximum file size, 0 = no limit")
	flag.IntVar(&Defaults.OldVersions, "logversions", Defaults.OldVersions, "Default old versions of file to keep (otherwise deleted)")
	flag.BoolVar(&NoStderr, "lognostderr", NoStderr, "Default to no logging to stderr")
	flag.IntVar(&Defaults.CheckSeconds, "logcheckseconds", Defaults.CheckSeconds, "Default seconds to check log file still exists")
	flag.IntVar(&Defaults.FlushSeconds, "logflushseconds", Defaults.FlushSeconds, "Default seconds to wait before flushing pending writes to the log file")

	if NoStderr {
		Defaults.Flags = FileOnly
	}
}

// LogFile implements an io.Writer so can used by the standard log library
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

	file        *os.File
	lastChecked time.Time
	size        int64
	messages    chan logMessage
	buf         *bufio.Writer
}

// New creates, if necessary, and opens a log file.
// If a LogFile is passed any empty fields are filled with suitable defaults.
// If nil is passed an empty LogFile is created and then filled in.
// Once finished with the LogFile call Close()
func New(lp *LogFile) (*LogFile, error) {
	if lp == nil {
		lp = new(LogFile)
		if lp == nil {
			return nil, fmt.Errorf("failed to create LogFile (out of memory?)")
		}
	}
	if lp.FileName == "" {
		if !defaultFileNameUsed {
			lp.FileName = Defaults.FileName
			// the logfile passed via the command line is only used once
			defaultFileNameUsed = true
		}
	}
	if lp.FileName == "" {
		return lp, fmt.Errorf("LogFile no file name")
	}
	if lp.FileMode == 0 {
		lp.FileMode = Defaults.FileMode
	}
	if lp.MaxSize == 0 {
		lp.MaxSize = Defaults.MaxSize
	}
	if lp.RotateFileFunc == nil {
		lp.RotateFileFunc = lp.RotateFileFuncDefault
	}
	if lp.CheckSeconds == 0 {
		lp.CheckSeconds = Defaults.CheckSeconds
	}
	if lp.FlushSeconds == 0 {
		lp.FlushSeconds = Defaults.FlushSeconds
	}
	if lp.Flags == 0 {
		if NoStderr {
			lp.Flags = FileOnly
		}
	}
	lp.messages = make(chan logMessage, logMessages)
	if lp.messages == nil {
		return nil, fmt.Errorf("LogFile failed to create channel (out of memory?)")
	}
	lp.messages <- logMessage{action: openLog}
	ready := make(chan (bool))
	go logger(lp, ready)
	if !<-ready {
		return lp, fmt.Errorf("LogFile failed to create file %s", lp.FileName)
	}

	return lp, nil
}

// Messages sent to the log handling goroutine: logger
type logMessage struct {
	action   logAction
	data     []byte
	complete chan<- bool
}

type logAction int

const (
	openLog logAction = iota
	writeLog
	rotateLog
	flushLog
	closeLog

	logMessages = 100
)

// logger loops until closeLog or an error happens handling log related actions.
// Once the log file is opened true is sent to the ready channel. If there
// if a problem opening the log file false is sent.
func logger(lp *LogFile, ready chan (bool)) {
	// flushChan will be nil unless FlushSeconds > 0
	// Note that a negative FlushSeconds is handled in writeLog
	var flushChan <-chan time.Time
	if lp.FlushSeconds > 0 {
		flushTicker := time.NewTicker(time.Second * time.Duration(lp.FlushSeconds))
		defer flushTicker.Stop()
		flushChan = flushTicker.C
	}

	// vanishChan will be nil unless CheckSeconds > 0
	var vanishChan <-chan time.Time
	if lp.CheckSeconds > 0 {
		vanishTicker := time.NewTicker(time.Second * time.Duration(lp.CheckSeconds))
		defer vanishTicker.Stop()
		vanishChan = vanishTicker.C
	}

	// Just in case... regularly check that this goroutine is still needed
	errorTicker := time.NewTicker(time.Second * time.Duration(errorSeconds))
	defer errorTicker.Stop()

	for {
		select {
		case message := <-lp.messages:
			switch message.action {
			case openLog:
				ready <- lp.startLog()
			case writeLog:
				lp.writeLog(message.data)
			case flushLog:
				lp.flushLog()
				message.complete <- true
			case rotateLog:
				lp.rotateLog()
			case closeLog:
				lp.closeLog()
				message.complete <- true
				return
			}
		case <-flushChan:
			lp.flushLog()
		case <-vanishChan:
			lp.vanishedLog()
		case <-errorTicker.C:
			if lp.file == nil {
				return
			}
		}
	}
}

// startLog creates or opens the log file. Depending on Flags the logfile may
// be rotated first. If all goes well startLog returns true.
// On a problem an error is printed to stderr (subject to the NoErrors flag)
// and false returned.
func (lp *LogFile) startLog() bool {
	if (lp.Flags&RotateOnStart) == RotateOnStart && lp.RotateFileFunc != nil {
		lp.RotateFileFunc()
	}

	truncated := lp.Flags&OverWriteOnStart == OverWriteOnStart

	return lp.openLogFile(truncated)
}

// openLogFile returns true if the file successfully opened and buffered.
// The truncated option will cause the file to be truncated on opening.
func (lp *LogFile) openLogFile(truncated bool) bool {
	lp.closeLog()

	var err error

	flags := os.O_RDWR | os.O_CREATE
	if truncated {
		flags = flags | os.O_TRUNC
	} else {
		flags = flags | os.O_APPEND
	}

	lp.file, err = os.OpenFile(lp.FileName, flags, lp.FileMode)
	if err != nil {
		lp.PrintError("LogFile failed to create %s: %s\n", lp.FileName, err)
		lp.file = nil
		return false
	}

	// Find the file size
	if truncated {
		lp.size = 0
	} else {
		fi, err := os.Stat(lp.FileName)
		if err == nil {
			lp.size = fi.Size()
		} else {
			lp.PrintError("LogFile unable to find initial filesize for %s: %s\n", lp.FileName, err)
			lp.size = 0
			// Hmmm... should I stop logging... no better to try and continue
			err = nil
		}
	}

	lp.buf = bufio.NewWriter(lp.file)
	if lp.buf == nil {
		lp.PrintError("LogFile error cannot create buffer for %s (out of memory?)\n", lp.FileName)
		lp.file.Close()
		lp.file = nil
		return false
	}

	return true
}

// writeLog writes p to stderr if required then writes it to the file.
// If writing to the file would cause the file to go over its size limit the file
// is closed, rotated (which may do nothing) and the opened with truncation.
func (lp *LogFile) writeLog(p []byte) {
	fileOnly := lp.Flags&FileOnly == FileOnly

	if !fileOnly {
		_, err := os.Stderr.Write(p)
		if err != nil {
			// Well I can't write to stderr to report it... so just return
			return
		}
	}

	if lp.file == nil {
		return
	}

	// Am I about to go over my file size limit?
	if lp.MaxSize > 0 && (lp.size+int64(len(p))) >= lp.MaxSize {
		lp.closeLog()

		if lp.RotateFileFunc != nil {
			lp.RotateFileFunc()
		}

		// Recreate the logfile truncating it (in case it wasn't rotated)
		if !lp.openLogFile(truncateLog) {
			return
		}
	}

	n, err := lp.buf.Write(p)
	if err != nil {
		lp.PrintError("Logfile error writing to %s: %s\n", lp.FileName, err)
	}
	if lp.FlushSeconds <= 0 {
		lp.flushLog()
	}

	lp.size += int64(n)

	return
}

// rotateLog closes the log file, calls the (possibly user) RotateFileFunc and
// reopens the log file
func (lp *LogFile) rotateLog() {
	if lp.RotateFileFunc == nil {
		return
	}
	lp.closeLog()
	lp.RotateFileFunc()
	lp.openLogFile(noTruncateLog)
}

// flushLog flushes out any pending writes to the log file
func (lp *LogFile) flushLog() {
	if lp.file == nil {
		return
	}

	err := lp.buf.Flush()
	if err != nil {
		lp.PrintError("LogFile error flushing %s: %s\n", lp.FileName, err)
	}
}

// vanishLog checks that the log file hasn't vanished.
// Perhaps it has been moved aside by something like Linux logrotate.
// If it has vanished then the log file is closed and reopened
func (lp *LogFile) vanishedLog() {
	_, err := os.Stat(lp.FileName)
	if err == nil {
		return
	}
	// Close and reopen the file
	lp.closeLog()
	lp.openLogFile(noTruncateLog)
}

// closeLog flushes and closes a log file
func (lp *LogFile) closeLog() {
	if lp.file == nil {
		return
	}

	lp.flushLog()

	err := lp.file.Close()
	if err != nil {
		lp.PrintError("LogFile error closing %s: %s\n", lp.FileName, err)
	}

	lp.file = nil
}

// PrintError prints out internal errors to standard error (if not turned off by the NoErrors flag)
func (lp *LogFile) PrintError(format string, args ...interface{}) {
	if lp.Flags&NoErrors == NoErrors {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// FileNameVersion returns a versioned log file name for rotating.
// If v is zero it returns the file name unmodified.
// Otherwise it add .v to the end
func FileNameVersion(fileName string, v int) string {
	if v == 0 {
		return fileName
	}
	return fmt.Sprintf("%s.%d", fileName, v)
}

// RotateFileFuncDefault only rotates if OldVersions non zero.
// It deletes the oldest version and renames the others log -> log.1, log.1 -> log.2...
func (lp *LogFile) RotateFileFuncDefault() {
	if lp.OldVersions <= 0 {
		return
	}

	// Delete the oldest
	oldFileName := FileNameVersion(lp.FileName, lp.OldVersions)
	_, err := os.Stat(oldFileName)
	if err == nil {
		err := os.Remove(oldFileName)
		if err != nil {
			lp.PrintError("LogFile error removing old file %s: %s\n", oldFileName, err)
		}
	}

	// Rename the others log -> log.1, log.1 -> log.2...
	for v := lp.OldVersions - 1; v >= 0; v-- {
		oldFilename := FileNameVersion(lp.FileName, v)
		olderFileName := FileNameVersion(lp.FileName, v+1)
		_, err = os.Stat(oldFilename)
		if err != nil {
			// Old file does not exist
			continue
		}
		err := os.Rename(oldFilename, olderFileName)
		if err != nil {
			lp.PrintError("LogFile error renaming old file %s to %s: %s\n", oldFilename, olderFileName, err)
		}
	}
}

// RotateFile requests an immediate file rotation.
func (lp *LogFile) RotateFile() {
	lp.messages <- logMessage{action: rotateLog}
}

// Flush writes any pending log entries out
func (lp *LogFile) Flush() {
	complete := make(chan bool)
	lp.messages <- logMessage{action: flushLog, complete: complete}
	<-complete
}

// Write is called by Log to write log entries.
func (lp *LogFile) Write(p []byte) (n int, err error) {
	// LogFile cannot guarantee that it will have finished with p before this
	// function returns. To prevent corruption use a copy of p.
	pLen := len(p)
	buf := make([]byte, pLen)
	copy(buf, p)

	lp.messages <- logMessage{action: writeLog, data: buf}
	if lp.FlushSeconds <= 0 {
		lp.Flush()
	}
	return pLen, nil
}

// Close flushs any pending data out and then closes a log file opened by calling New()
func (lp *LogFile) Close() {
	complete := make(chan bool)
	lp.messages <- logMessage{action: closeLog, complete: complete}
	// wait for the logfile to close
	<-complete
}
