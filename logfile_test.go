/*
File summary: logfile testing
Package: logfile
Author: Lee McLoughlin

Copyright (C) 2015 LMMR Tech Ltd
*/

package logfile

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	tmpDir    = "/tmp"
	tmpPrefix = "lftest"
	showDebug = true
)

// Return a unique filename or an error
func tempFileName() (string, error) {
	f, err := ioutil.TempFile(tmpDir, tmpPrefix)
	if err != nil {
		return "", err
	}
	f.Close()
	debug("tempFileName " + f.Name())
	return f.Name(), nil
}

func debug(msg string) {
	if !showDebug {
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

func Test_FilenameVersions(t *testing.T) {
	debug("Test_Filenames start")
	defer debug("Test_Filenames end")

	logFileName := "example.log"
	v0 := FileNameVersion(logFileName, 0)
	if logFileName != v0 {
		t.Errorf("FileNameVersion wrong expected %s got %s", logFileName, v0)
	}
	v1 := FileNameVersion(logFileName, 1)
	lfv1 := logFileName + ".1"
	if lfv1 != v1 {
		t.Errorf("FileNameVersion wrong expected %s got %s", lfv1, v1)
	}
}

func Test_DefaultCreate(t *testing.T) {
	debug("Test_DefaultCreate start")
	defer debug("Test_DefaultCreate end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	// Pretend the -logfile flag was used
	logfile = logFileName

	logFile, err := New(nil)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)

	msg := "hello\n"
	log.Print(msg)
	logFile.Flush()

	fi, err := os.Stat(logFileName)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	logFile.Close()

	contents, err := ioutil.ReadFile(logFileName)
	if err != nil {
		t.Errorf("Failed to read log file %s: %s\n", logFileName, err)
		return
	}

	size := int64(len(msg))
	if fi.Size() != size {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, size, fi.Size())
	} else if string(contents) != msg {
		t.Errorf("Wrong logfile contents for %s expected %s got %d\n", logFileName, msg, fi.Size())
	} else {
		t.Log("Log file created and has correct size and contents")
	}

	os.Remove(logFileName)
}

func Test_BigMessages(t *testing.T) {
	debug("Test_BigMessages start")
	defer debug("Test_BigMessages end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	// Flush and check every second
	// Naughty: set the internall error check timer
	errorSeconds = 1
	logFile, err := New(&LogFile{
		FileName:     logFileName,
		FlushSeconds: 1,
		CheckSeconds: 1,
		Flags:        FileOnly})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)

	msg := ""
	for i := 0; i < 5; i++ {
		line := strings.Repeat(string('0'+i), 70) + "\n"
		log.Print(line)
		msg = msg + line
	}
	size := int64(len(msg))

	// Check for log being writen
	for i := 0; i < 10; i++ {
		fi, err := os.Stat(logFileName)
		if err != nil {
			t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
			return
		}
		if fi.Size() == size {
			break
		}
		t.Logf("Log size after %d seconds %d", i, fi.Size())
		time.Sleep(time.Second)
	}

	// Wait a bit so flush and check are run
	time.Sleep(time.Second * 3)
	logFile.Close()

	fi, err := os.Stat(logFileName)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	contents, err := ioutil.ReadFile(logFileName)
	if err != nil {
		t.Errorf("Failed to read log file %s: %s\n", logFileName, err)
		return
	}

	if fi.Size() != size {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, size, fi.Size())
	} else if string(contents) != msg {
		t.Errorf("Wrong logfile contents for %s expected %s got %d\n", logFileName, msg, fi.Size())
	} else {
		t.Log("Log file created and has correct size and contents")
	}

	os.Remove(logFileName)
}

func Test_Rotation(t *testing.T) {
	debug("Test_Rotation start")
	defer debug("Test_Rotation end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	logFile, err := New(&LogFile{
		FileName:    logFileName,
		MaxSize:     71, // Same as size of test lines below
		OldVersions: 2,
		Flags:       FileOnly})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)

	msg := ""
	for i := 0; i < 5; i++ {
		line := strings.Repeat(string('0'+i), 70) + "\n"
		log.Print(line)
		msg = msg + line
	}
	logFile.Close()

	oldest := 4
	for i := 0; i < 3; i++ {
		lf := FileNameVersion(logFileName, i)
		fi, err := os.Stat(lf)
		if err != nil {
			t.Errorf("Failed to create log file %s: %s\n", lf, err)
			return
		}

		contents, err := ioutil.ReadFile(lf)
		if err != nil {
			t.Errorf("Failed to read log file %s: %s\n", lf, err)
			return
		}

		line := strings.Repeat(string('0'+oldest), 70) + "\n"
		oldest--

		size := int64(len(line))
		if fi.Size() != size {
			t.Errorf("Wrong logfile size for %s expected %d got %d\n", lf, size, fi.Size())
		} else if string(contents) != line {
			t.Errorf("Wrong logfile contents for %s expected %s got %s\n", lf, line, contents)
		} else {
			t.Logf("Log file %s created and has correct size and contents", lf)
			os.Remove(lf)
		}
	}
}

func Test_ExplicitRotation(t *testing.T) {
	debug("Test_ExplicitRotation start")
	defer debug("Test_ExplicitRotation end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	// Flush and check every second
	logFile, err := New(&LogFile{
		FileName:    logFileName,
		OldVersions: 2,
		Flags:       FileOnly})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)

	line := strings.Repeat(string('0'), 70) + "\n"
	log.Print(line)

	t.Logf("Forcing rotation")
	logFile.RotateFile()

	line = strings.Repeat(string('1'), 70) + "\n"
	log.Print(line)

	logFile.Close()

	oldest := 1
	for i := 0; i < 2; i++ {
		lf := FileNameVersion(logFileName, i)
		fi, err := os.Stat(lf)
		if err != nil {
			t.Errorf("Failed to create log file %s: %s\n", lf, err)
			return
		}

		contents, err := ioutil.ReadFile(lf)
		if err != nil {
			t.Errorf("Failed to read log file %s: %s\n", lf, err)
			return
		}

		line := strings.Repeat(string('0'+oldest), 70) + "\n"
		oldest--

		size := int64(len(line))
		if fi.Size() != size {
			t.Errorf("Wrong logfile size for %s expected %d got %d\n", lf, size, fi.Size())
		} else if string(contents) != line {
			t.Errorf("Wrong logfile contents for %s expected %s got %s\n", lf, line, contents)
		} else {
			t.Logf("Log file %s created and has correct size and contents", lf)
			os.Remove(lf)
		}
	}
}

func Test_OverWriteOnStart(t *testing.T) {
	debug("Test_OverWriteOnStart start")
	defer debug("Test_OverWriteOnStart end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	f, err := os.Create(logFileName)
	if err != nil {
		t.Errorf("Failed to create file %s: %s\n", logFileName, err)
		return
	}
	fmt.Fprintf(f, "I AM GOING TO BE OVERWRITEN\n")
	f.Close()

	logFile, err := New(&LogFile{FileName: logFileName, Flags: OverWriteOnStart})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetOutput(logFile)
	logFile.Close()

	fi, err := os.Stat(logFileName)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	if fi.Size() != 0 {
		t.Errorf("Wrong logfile size for %s expected 0 got %d\n", logFileName, fi.Size())
	} else {
		t.Log("Log file created and has correct size")
	}

	os.Remove(logFileName)
}

func Test_LogVanish(t *testing.T) {
	debug("Test_LogVanish start")
	defer debug("Test_LogVanish end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	logFile, err := New(&LogFile{
		FileName:     logFileName,
		CheckSeconds: 1,
		Flags:        OverWriteOnStart})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetOutput(logFile)
	log.Print("testing")

	// Remove the logfile and wait long enough for LogFile to notice (CheckSeconds is 1)
	os.Remove(logFileName)
	time.Sleep(time.Second * 2)

	// This is all that should appear in the logfile
	msg := "testing again\n"
	log.Print(msg)

	logFile.Close()

	fi, err := os.Stat(logFileName)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	if fi.Size() != int64(len(msg)) {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, len(msg), fi.Size())
	} else {
		t.Log("Log file created and has correct size")
	}

	os.Remove(logFileName)
}

func Test_AppendOnStart(t *testing.T) {
	debug("Test_AppendOnStart start")
	defer debug("Test_AppendOnStart end")

	logFileName, err := tempFileName()
	if err != nil {
		t.Errorf("Failed to create temporary file: %s\n", err)
		return
	}

	contents := "I AM GOING TO BE APPENDED TO\n"

	f, err := os.Create(logFileName)
	if err != nil {
		t.Errorf("Failed to create file %s: %s\n", logFileName, err)
		return
	}
	fmt.Fprint(f, contents)
	f.Close()

	logFile, err := New(&LogFile{FileName: logFileName})
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)

	fi, err := os.Stat(logFileName)
	if err != nil {
		t.Errorf("Failed to create log file %s: %s\n", logFileName, err)
		return
	}

	size := int64(len(contents))
	if fi.Size() != size {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, size, fi.Size())
	} else {
		t.Log("Log file created and has correct size")
	}

	os.Remove(logFileName)
}

func ExampleLogFile() {
	debug("ExampleLogFile start")
	defer debug("ExampleLogFile end")

	logFileName, err := tempFileName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temporary file: %s\n", err)
		return
	}

	logFile, err := New(
		&LogFile{
			FileName: logFileName,
			MaxSize:  500 * 1024,
			Flags:    OverWriteOnStart})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file %s: %s\n", logFileName, err)
		os.Exit(1)
	}

	log.SetOutput(logFile)
	log.Print("hello")
	logFile.Close()

	writen, err := ioutil.ReadFile(logFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read log file %s: %s\n", logFileName, err)
		os.Exit(1)
	}
	fmt.Print(string(writen))
	// Output: hello

	os.Remove(logFileName)
}
