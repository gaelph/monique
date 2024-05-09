package mediator

type MediatorListener interface {
	OnStart(command string)
	OnError(err error)
	OnKill()
	OnStop()
	OnOutput(output string)
	OnRequestRestart()
}

type Mediator interface {
	SendStart(command string)
	SendError(err error)
	SendKill()
	SendStop()
	SendOutput(output string)
	SendRequestRestart()
	AddListener(listener MediatorListener)
}

type mediator struct {
	listeners []MediatorListener
}

func NewMediator() Mediator {
	return &mediator{}
}

func (mediator *mediator) AddListener(listener MediatorListener) {
	mediator.listeners = append(mediator.listeners, listener)
}

func (mediator *mediator) SendStart(command string) {
	for _, listener := range mediator.listeners {
		listener.OnStart(command)
	}
}

func (mediator *mediator) SendError(err error) {
	for _, listener := range mediator.listeners {
		listener.OnError(err)
	}
}

func (mediator *mediator) SendKill() {
	for _, listener := range mediator.listeners {
		listener.OnKill()
	}
}

func (mediator *mediator) SendStop() {
	for _, listener := range mediator.listeners {
		listener.OnStop()
	}
}

func (mediator *mediator) SendOutput(output string) {
	for _, listener := range mediator.listeners {
		listener.OnOutput(output)
	}
}

func (mediator *mediator) SendRequestRestart() {
	for _, listener := range mediator.listeners {
		listener.OnRequestRestart()
	}
}
