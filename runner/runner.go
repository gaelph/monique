package runner

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/gaelph/monique/mediator"
)

type Runner struct {
	mediator         mediator.Mediator
	control          chan bool
	debouncedRestart func()
	Command          []string
	delay            int
}

func NewRunner(command []string, delay int) *Runner {
	r := &Runner{
		Command: command,
		control: make(chan bool),
		delay:   delay,
	}

	r.debouncedRestart = debounce(func() {
		r.restart()
	}, 150)

	return r
}

func (r *Runner) SetMediator(mediator mediator.Mediator) {
	r.mediator = mediator
	r.mediator.AddListener(r)
}

func (r *Runner) Start() {
	log.Println("Starting process")
	time.Sleep(time.Duration(r.delay) * time.Millisecond)

	if r.mediator != nil {
		r.mediator.SendStart(strings.Join(r.Command, " "))
	}

	cmdName := r.Command[0]
	cmdArgs := r.Command[1:]
	cmd := exec.CommandContext(context.Background(), cmdName, cmdArgs...)
	t, err := pty.Start(cmd)
	if err != nil {
		if r.mediator != nil {
			r.mediator.SendError(err)
		}
		return
	}

	go func(r *Runner) {
		for {
			bytes := make([]byte, 1024)
			n, err := t.Read(bytes)
			// r.Output <- string(bytes[:n])
			go r.mediator.SendOutput(string(bytes[:n]))

			if err != nil {
				if r.mediator != nil {
					r.mediator.SendStop()
				}
				return
			}

		}
	}(r)

	defer func() {
		t.Close()
	}()

	<-r.control
	cmd.Process.Signal(syscall.SIGTERM)
	if r.mediator != nil {
		r.mediator.SendKill()
	}
}

func (r *Runner) Stop() {
	log.Println("Killing process")
	go r.mediator.SendOutput("Killing process\n")
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

// MARK: - MediatorListener

func (runner *Runner) OnStart(command string) {
}

func (runner *Runner) OnError(err error) {
}

func (runner *Runner) OnKill() {
}

func (runner *Runner) OnStop() {
}

func (runner *Runner) OnOutput(output string) {
}

func (runner *Runner) OnRequestRestart() {
	runner.debouncedRestart()
}
