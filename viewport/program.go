package viewport

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gaelph/monique/runner"
)

type Program struct {
	prog *tea.Program
}

func NewProgram() *Program {
	return &Program{
		prog: tea.NewProgram(
			NewModel(),
			tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
			tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
		),
	}
}

func (p *Program) Append(content string) {
	p.prog.Send(AppendContentMsg{Content: content})
}

func (p *Program) Run() {
	if _, err := p.prog.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}

// MARK: RunnerDelegate

func (p *Program) OnStart(r runner.Runner) {
	p.prog.Send(ClearContentMsg{})
	p.prog.Send(AppendContentMsg{Content: fmt.Sprintf("Starting %s\n", strings.Join(r.Command, " "))})
}

func (p *Program) OnError(r runner.Runner, err error) {
	p.prog.Send(AppendContentMsg{Content: fmt.Sprintf("Error: %s\n", err)})
}

func (p *Program) OnKill(r runner.Runner) {
	p.prog.Send(AppendContentMsg{Content: "Killing process\n"})
}

func (p *Program) OnStop(r runner.Runner) {
	p.prog.Send(AppendContentMsg{Content: "Process exited\n"})
}

func (p *Program) OnOutput(r runner.Runner, output string) {
	p.prog.Send(AppendContentMsg{Content: output})
}
