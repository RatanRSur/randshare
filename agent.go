package main

import (
	_ "fmt"
)

type Agent struct {
	inbox chan string
	peers []*Agent
}

func newAgent() *Agent {
	inbox := make(chan string, 10)
	return &Agent{inbox, []*Agent{}}
}

func (a *Agent) providePeers(agents []*Agent) {
	for _, agent := range agents {
		if agent != a {
			a.peers = append(a.peers, agent)
		}
	}
}

func (a Agent) run() {
	for {
		message := <-a.inbox
		switch message {
		}
	}
}

func (a Agent) tell(message string) {
	a.inbox <- message
}

func main() {
	var agents []*Agent
	for i := 0; i < 5; i++ {
		agents = append(agents, newAgent())
	}
	for _, agent := range agents {
		agent.providePeers(agents)
	}
	for _, agent := range agents {
		agent.run()
	}
}
