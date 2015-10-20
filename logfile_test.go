// logfile_test.go
package logfile

import (
	"io/ioutil"
	"fmt"
	"log"
	"os"
	"testing"
)

func Test_DefaultCreate(t *testing.T) {
	logFileName := "example.log"

	os.Remove(logFileName)

	logFile, err := New(&LogFile{FileName: logFileName, FlushSeconds: -1})
	if err != nil {
		t.Errorf("Failed to create log plus %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)
	
	msg := "hello\n"
	log.Print(msg)
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

    size := int64(len(msg))
	if fi.Size() != size {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, size, fi.Size())
	} else if string(contents) != msg {
		t.Errorf("Wrong logfile contents for %s expected %s got %d\n", logFileName, msg, fi.Size())
	} else {
		t.Log("Log Plus created and has correct size and contents")
		os.Remove(logFileName)
	}
}

func Test_BigMessages(t *testing.T) {
	logFileName := "example.log"

	os.Remove(logFileName)

	logFile, err := New(&LogFile{FileName: logFileName, FlushSeconds: -1})
	if err != nil {
		t.Errorf("Failed to create log plus %s: %s\n", logFileName, err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(logFile)
	
	m := "I am a very long log message line to test that writing long lines to log files works\n"
	msg := ""
	for i := 0; i < 5; i++ {
		msg = msg + m
	}
	log.Print(msg)
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

    size := int64(len(msg))
	if fi.Size() != size {
		t.Errorf("Wrong logfile size for %s expected %d got %d\n", logFileName, size, fi.Size())
	} else if string(contents) != msg {
		t.Errorf("Wrong logfile contents for %s expected %s got %d\n", logFileName, msg, fi.Size())
	} else {
		t.Log("Log Plus created and has correct size and contents")
		os.Remove(logFileName)
	}
}

func Test_OverWriteOnStart(t *testing.T) {
	logFileName := "example.log"

	os.Remove(logFileName)

	f, err := os.Create(logFileName)
	if err != nil {
		t.Errorf("Failed to create file %s: %s\n", logFileName, err)
		return
	}
	fmt.Fprintf(f, "I AM GOING TO BE OVERWRITEN\n")
	f.Close()

	logFile, err := New(&LogFile{FileName: logFileName, Flags: OverWriteOnStart})
	if err != nil {
		t.Errorf("Failed to create log plus %s: %s\n", logFileName, err)
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
		t.Log("Log Plus created and has correct size")
	}

	os.Remove(logFileName)
}

func Test_AppendOnStart(t *testing.T) {
	logFileName := "example.log"

	os.Remove(logFileName)
	
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
		t.Errorf("Failed to create log plus %s: %s\n", logFileName, err)
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
		t.Log("Log Plus created and has correct size")
	}
	
	os.Remove(logFileName)
}

func ExampleLogFile() {
	logFileName := "example.log"
	logFile, err := New(
		&LogFile{
			FileName:     logFileName,
			Flags:        OverWriteOnStart})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log plus %s: %s\n", logFileName, err)
		os.Exit(1)
	}

	log.SetOutput(logFile)
	log.Print("hello")
	logFile.Close()
}
