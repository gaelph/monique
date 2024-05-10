package viewport

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Blur          key.Binding
	Search        key.Binding
	Filter        key.Binding
	Accept        key.Binding
	NextMatch     key.Binding
	PreviousMatch key.Binding
	Quit          key.Binding
	HalfPageUp    key.Binding
	HalfPageDown  key.Binding
	Restart       key.Binding
	ShowHelp      key.Binding
}

func DefaultKeyBinding() KeyMap {
	return KeyMap{
		Blur:          key.NewBinding(key.WithKeys("esc")),
		Search:        key.NewBinding(key.WithKeys("/")),
		Filter:        key.NewBinding(key.WithKeys("f")),
		Accept:        key.NewBinding(key.WithKeys("enter")),
		NextMatch:     key.NewBinding(key.WithKeys("n")),
		PreviousMatch: key.NewBinding(key.WithKeys("N")),
		Quit:          key.NewBinding(key.WithKeys("ctrl+c")),
		HalfPageUp:    key.NewBinding(key.WithKeys("ctrl+u")),
		HalfPageDown:  key.NewBinding(key.WithKeys("ctrl+d")),
		Restart:       key.NewBinding(key.WithKeys("ctrl+r")),
		ShowHelp:      key.NewBinding(key.WithKeys("?")),
	}
}
