package runner

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type RunnerDelegate interface {
	OnStart(r Runner)
	OnError(r Runner, err error)
	OnKill(r Runner)
	OnStop(r Runner)
	OnOutput(r Runner, output string)
}

type Runner struct {
	Delegate         RunnerDelegate
	control          chan bool
	Output           chan string
	debouncedRestart func()
	Command          []string
	delay            int
}

func NewRunner(command []string, delay int) *Runner {
	r := &Runner{
		Command: command,
		control: make(chan bool),
		delay:   delay,
		Output:  make(chan string),
	}

	r.debouncedRestart = debounce(func() {
		r.restart()
	}, 150)

	return r
}

func (r *Runner) Start() {
	time.Sleep(time.Duration(r.delay) * time.Millisecond)

	if r.Delegate != nil {
		r.Delegate.OnStart(*r)
	}

	cmdName := r.Command[0]
	cmdArgs := r.Command[1:]
	cmd := exec.CommandContext(context.Background(), cmdName, cmdArgs...)
	t, err := pty.Start(cmd)
	if err != nil {
		if r.Delegate != nil {
			r.Delegate.OnError(*r, err)
		}
		return
	}

	go func(out chan string) {
		for {
			bytes := make([]byte, 128)
			n, err := t.Read(bytes)
			out <- string(bytes[:n])

			if err != nil {
				if r.Delegate != nil {
					r.Delegate.OnStop(*r)
				}
				return
			}

		}
	}(r.Output)

	defer func() {
		t.Close()
	}()

	<-r.control
	cmd.Process.Signal(syscall.SIGTERM)
	if r.Delegate != nil {
		r.Delegate.OnKill(*r)
	}
}

func (r *Runner) Stop() {
	r.Output <- "Killing process\n"
	r.control <- true
}

func (r *Runner) Restart() {
	r.debouncedRestart()
}

func (r *Runner) restart() {
	r.Stop()
	time.Sleep(time.Duration(100) * time.Millisecond)
	go r.Start()
}

func (r *Runner) SetDelegate(delegate RunnerDelegate) {
	r.Delegate = delegate

	go func(d RunnerDelegate) {
		for {
			string := <-r.Output
			d.OnOutput(*r, string)
		}
	}(r.Delegate)
}

// a debounce function and struct
func debounce(f func(), d int) func() {
	var timer *time.Timer
	return func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(time.Duration(d)*time.Millisecond, func() {
			f()
		})
	}
}
