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
	}
}
