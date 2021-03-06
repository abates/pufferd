package shutdown

import (
	"github.com/pufferpanel/pufferd/programs"
	"github.com/pufferpanel/pufferd/logging"
	"github.com/braintree/manners"
	"os/signal"
	"os"
	"runtime/debug"
	"sync"
	"github.com/pkg/errors"
	"time"
	"syscall"
)

func CompleteShutdown() {
	logging.Warn("Interrupt received, stopping servers")
	wg := Shutdown()
	wg.Wait()
	logging.Info("All servers stopped")
	os.Exit(0)
}

func Shutdown() *sync.WaitGroup {
	defer func() {
		if err := recover(); err != nil {
			logging.Errorf("Error: %+v\n%s", err, debug.Stack())
		}
	}()
	wg := sync.WaitGroup{}
	manners.Close()
	prgs := programs.GetAll()
	wg.Add(len(prgs))
	for _, element := range prgs {
		go func(e programs.Program) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					logging.Errorf("Error: %+v\n%s", err, debug.Stack())
				}
			}()
			logging.Warn("Stopping program " + e.Id())
			err := e.Stop()
			if err != nil {
				logging.Error("Error stopping server " + e.Id(), err)
			}
			err = e.GetEnvironment().WaitForMainProcess()
			if err != nil {
				logging.Error("Error stopping server " + e.Id(), err)
			}
			logging.Warn("Stopped program " + e.Id())
		}(element)
	}
	return &wg;
}

func CreateHook() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logging.Errorf("Error: %+v\n%s", err, debug.Stack())
			}
		}()
		<- c
		CompleteShutdown()
	}()
}

func Command(pid int) {
	proc, err := os.FindProcess(pid);
	if err != nil || proc == nil {
		if err == nil && proc == nil {
			err = errors.New("no process found")
		}
		logging.Error("Error shutting down pufferd", err)
		return
	}
	err = proc.Signal(os.Interrupt)
	if err != nil {
		logging.Error("Error shutting down pufferd", err)
		return
	}

	wait := make(chan error)

	waitForProcess(proc, wait)

	err = <- wait

	if err != nil {
		logging.Error("Error shutting down pufferd", err)
		return
	}
	err = proc.Release()
	if err != nil {
		logging.Error("Error shutting down pufferd", err)
		return
	}
}

func waitForProcess(process *os.Process, c chan error) {
	var err error
	timer := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range timer.C {
			err = process.Signal(syscall.Signal(0))
			if err != nil {
				if err.Error() == "os: process already finished" {
					c <- nil
				} else {
					c <- err
				}

				timer.Stop()
			} else {
			}
		}
	}()
}