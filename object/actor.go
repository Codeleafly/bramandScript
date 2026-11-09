package object

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	actorCounter    uint64
	actorRegistry   = make(map[string]*Actor)
	actorRegistryMu sync.RWMutex
)

// RegisterActor adds an actor to the global process registry by name.
func RegisterActor(name string, act *Actor) bool {
	actorRegistryMu.Lock()
	defer actorRegistryMu.Unlock()
	actorRegistry[name] = act
	act.Mu.Lock()
	act.Name = name
	act.Mu.Unlock()
	return true
}

// WhereisActor looks up an actor by registered name.
func WhereisActor(name string) *Actor {
	actorRegistryMu.RLock()
	defer actorRegistryMu.RUnlock()
	return actorRegistry[name]
}

// UnregisterActor removes an actor from the registry.
func UnregisterActor(name string) {
	actorRegistryMu.Lock()
	defer actorRegistryMu.Unlock()
	delete(actorRegistry, name)
}

// ClearActorRegistry resets the actor registry (used for tests).
func ClearActorRegistry() {
	actorRegistryMu.Lock()
	defer actorRegistryMu.Unlock()
	actorRegistry = make(map[string]*Actor)
}

type ActorMessage struct {
	Payload Object
	ReplyTo chan Object
}

type ActorRunner func(handler Object, args []Object) (Object, error)

type Actor struct {
	ID        string
	Name      string
	Mailbox   chan *ActorMessage
	Handler   Object
	State     Object
	Status    string // "running", "stopped", "failed"
	LastError string
	StopChan  chan struct{}
	Mu        sync.RWMutex
	Runner    ActorRunner
}

func (a *Actor) Type() ObjectType { return ACTOR_OBJ }

func (a *Actor) Inspect() string {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	if a.Name != "" {
		return fmt.Sprintf("<actor %s id=%s status=%s>", a.Name, a.ID, a.Status)
	}
	return fmt.Sprintf("<actor id=%s status=%s>", a.ID, a.Status)
}

// NewActor creates and spawns a new Actor process.
func NewActor(handler Object, initialState Object, runner ActorRunner) *Actor {
	idNum := atomic.AddUint64(&actorCounter, 1)
	id := fmt.Sprintf("act_%d", idNum)

	if initialState == nil {
		initialState = &Null{}
	}

	act := &Actor{
		ID:       id,
		Mailbox:  make(chan *ActorMessage, 256),
		Handler:  handler,
		State:    initialState,
		Status:   "running",
		StopChan: make(chan struct{}),
		Runner:   runner,
	}

	go act.loop()
	return act
}

func (a *Actor) loop() {
	for {
		select {
		case <-a.StopChan:
			a.Mu.Lock()
			a.Status = "stopped"
			a.Mu.Unlock()
			return

		case msg, ok := <-a.Mailbox:
			if !ok {
				a.Mu.Lock()
				a.Status = "stopped"
				a.Mu.Unlock()
				return
			}
			a.handleMessage(msg)
		}
	}
}

func (a *Actor) handleMessage(msg *ActorMessage) {
	defer func() {
		if r := recover(); r != nil {
			a.Mu.Lock()
			a.Status = "failed"
			a.LastError = fmt.Sprintf("actor panic: %v", r)
			a.Mu.Unlock()
			if msg.ReplyTo != nil {
				msg.ReplyTo <- &Error{Message: a.LastError}
			}
		}
	}()

	a.Mu.RLock()
	handler := a.Handler
	state := a.State
	runner := a.Runner
	a.Mu.RUnlock()

	var res Object
	var err error

	// If handler accepts 2 arguments, pass (payload, state)
	numParams := 1
	if cl, ok := handler.(*Closure); ok && cl.Fn != nil {
		numParams = cl.Fn.NumParameters
	}

	if runner != nil {
		if numParams >= 2 {
			res, err = runner(handler, []Object{msg.Payload, state})
		} else {
			res, err = runner(handler, []Object{msg.Payload})
		}
	} else if b, ok := handler.(*Builtin); ok {
		if numParams >= 2 {
			res = b.Fn(msg.Payload, state)
		} else {
			res = b.Fn(msg.Payload)
		}
	} else {
		err = fmt.Errorf("actor handler is not callable: %s", handler.Type())
	}

	if err != nil {
		a.Mu.Lock()
		a.Status = "failed"
		a.LastError = err.Error()
		a.Mu.Unlock()
		if msg.ReplyTo != nil {
			msg.ReplyTo <- &Error{Message: err.Error()}
		}
		return
	}

	if res != nil && res.Type() != NULL_OBJ && numParams >= 2 {
		a.Mu.Lock()
		a.State = res
		a.Mu.Unlock()
	}

	if msg.ReplyTo != nil {
		if res == nil {
			res = &Null{}
		}
		msg.ReplyTo <- res
	}
}

// Send queues a message asynchronously into the actor's FIFO mailbox.
func (a *Actor) Send(payload Object) bool {
	a.Mu.RLock()
	status := a.Status
	a.Mu.RUnlock()

	if status != "running" {
		return false
	}

	select {
	case a.Mailbox <- &ActorMessage{Payload: payload}:
		return true
	default:
		// Mailbox full, queue in background to prevent caller deadlock
		go func() {
			a.Mailbox <- &ActorMessage{Payload: payload}
		}()
		return true
	}
}

// Ask sends a message synchronously and waits for a reply with timeout.
func (a *Actor) Ask(payload Object, timeout time.Duration) (Object, error) {
	a.Mu.RLock()
	status := a.Status
	a.Mu.RUnlock()

	if status != "running" {
		return nil, fmt.Errorf("actor is not running (status: %s)", status)
	}

	replyChan := make(chan Object, 1)
	msg := &ActorMessage{
		Payload: payload,
		ReplyTo: replyChan,
	}

	a.Mailbox <- msg

	select {
	case res := <-replyChan:
		return res, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("ask timeout after %v", timeout)
	}
}

// Stop terminates the actor loop.
func (a *Actor) Stop() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Status == "running" {
		a.Status = "stopped"
		close(a.StopChan)
	}
}

// Restart resets state and restarts processing loop if stopped or failed.
func (a *Actor) Restart(newState Object) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	if a.Status != "running" {
		if newState != nil {
			a.State = newState
		}
		a.Status = "running"
		a.LastError = ""
		a.StopChan = make(chan struct{})
		go a.loop()
	}
}
