package lamanager

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"github.com/tehsphinx/concurrent"
	"github.com/tehsphinx/dbg"
)

// RunProcess runs a Local Adapter process
func RunProcess() *Process {
	p := &Process{
		logBuffer: make([][]byte, 0, 100),
		running:   concurrent.NewBool(),
	}
	p.run()
	return p
}

type Process struct {
	logBuffer [][]byte
	logMutex  sync.RWMutex
	cmd       *exec.Cmd
	running   *concurrent.Bool
}

func (s *Process) run() {
	go func() {

		outfile, err := os.OpenFile("LocalAdapter.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		execPath := filepath.Join(getPath(), LAExecutable)

		astilog.Infof("Starting %s", execPath)
		s.cmd = exec.Command(execPath)
		if err != nil {
			dbg.Red(err)
			astilog.Debug(err)
		} else {
			defer outfile.Close()
			stdOut, err := s.cmd.StdoutPipe()
			if err != nil {
				astilog.Error(errors.Wrap(err, "could not hook to stdOut of local adapter"))
			}
			stdErr, err := s.cmd.StderrPipe()
			if err != nil {
				astilog.Error(errors.Wrap(err, "could not hook to stdOut of local adapter"))
			}
			//cmd.Stderr = outfile
			//cmd.Stdout = outfile
			//s.cmd.Stderr = os.Stderr
			//s.cmd.Stdout = os.Stdout

			s.writeOutput(stdOut, stdErr, outfile)
		}

		if r := s.cmd.Start(); r != nil {
			dbg.Red(r)
		}
		s.running.Set(true)
		defer s.running.Set(false)

		s.cmd.Wait()
	}()
}

func (s *Process) writeOutput(stdOut, stdErr io.ReadCloser, outfile *os.File) {
	go func() {
		sc := bufio.NewScanner(stdOut)
		for sc.Scan() {
			bytes := sc.Bytes()
			s.writeBuffer(bytes)

			line := append(bytes, '\n')
			if _, err := os.Stdout.Write(line); err != nil {
				log.Println("could not write to os.Stdout:", err)
			}

			if _, err := outfile.Write(line); err != nil {
				log.Println("could not write to log file:", err)
			}
		}
	}()

	go func() {
		sc := bufio.NewScanner(stdErr)
		for sc.Scan() {
			bytes := sc.Bytes()
			s.writeBuffer(bytes)

			line := append(bytes, '\n')
			if _, err := os.Stderr.Write(line); err != nil {
				log.Println("could not write to os.Stderr:", err)
			}

			if _, err := outfile.Write(line); err != nil {
				log.Println("could not write to log file:", err)
			}
		}
	}()
}

func (s *Process) writeBuffer(b []byte) {
	s.logMutex.Lock()
	if 95 < len(s.logBuffer) {
		// remove an item to make space
		s.logBuffer = s.logBuffer[1:]
	}
	s.logBuffer = append(s.logBuffer, b)
	s.logMutex.Unlock()
}

func (s *Process) GetLog() []string {
	s.logMutex.RLock()
	log := make([]string, 0, 100)
	for _, line := range s.logBuffer {
		log = append(log, string(line))
	}
	s.logMutex.RUnlock()
	return log
}

func (s *Process) IsRunning() bool {
	return s.running.Get()
}

func (s *Process) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Kill()
}

func getPath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	f := filepath.Dir(ex)
	// TODO: do no commit!!
	f = filepath.Join(os.Getenv("GOPATH"), "src", "git.raceresult.com", "LocalAdapterServer", "gui")
	dbg.Green(f)
	return f
}
