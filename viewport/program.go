package viewport

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gaelph/monique/mediator"
)

type Program struct {
	prog     *tea.Program
	mediator mediator.Mediator
}

func NewProgram(command string, mediator mediator.Mediator) *Program {
	prog := &Program{
		prog: tea.NewProgram(
			NewModel(command, mediator),
			tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
			tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
		),
		mediator: mediator,
	}

	mediator.AddListener(prog)

	return prog
}

func (p *Program) Append(content string) {
	p.prog.Send(AppendContentMsg{Content: content})
}

func (p *Program) Run() {
	f, err := tea.LogToFile("monique.log", "debug")
	if err != nil {
		log.Println("could not log to file:", err)
		os.Exit(1)
	}

	defer f.Close()

	if _, err := p.prog.Run(); err != nil {
		log.Println("could not run program:", err)
		os.Exit(1)
	}
}

// MARK: MediatorListener

func (p *Program) OnStart(command string) {
	p.prog.Send(ClearContentMsg{})
	p.prog.Send(AppendContentMsg{Content: fmt.Sprintf("Starting %s\n", command)})
}

func (p *Program) OnError(err error) {
	p.prog.Send(AppendContentMsg{Content: fmt.Sprintf("Error: %s\n", err)})
}

func (p *Program) OnKill() {
	p.prog.Send(AppendContentMsg{Content: "Killing process\n"})
}

func (p *Program) OnStop() {
	p.prog.Send(AppendContentMsg{Content: "Process exited\n"})
}

func (p *Program) OnOutput(output string) {
	p.prog.Send(AppendContentMsg{Content: output})
}

func (p *Program) OnRequestRestart() {
}
