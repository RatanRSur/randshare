package main

import (
	_ "fmt"
	"math/rand"
	"sync"
)

const numberOfAgents = 10
const t = 4
const q int = 2166136261

type Agent struct {
	inbox chan Message
	index int
	peers []*Agent
	shares []int
}

type MessageType uint8
const (
	SecretShare MessageType = iota
)

type Message struct {
	Type MessageType
	From int
	Value int
}

func pow(x int, y int) int {
	result := 1
	for i := 0; i < y; i++ {
		result *= x
	}
	return result
}

func evaluatePolynomial(coeffs []int, x int) int {
	result := 0
	for i, coeff := range coeffs {
		result += coeff * pow(x, i)
	}
	return result
}

func newAgent(index int) *Agent {
	inbox := make(chan Message, numberOfAgents * numberOfAgents)
	return &Agent{inbox, index, []*Agent{}, []int{}}
}

func (a *Agent) providePeers(agents []*Agent) {
	for _, agent := range agents {
		if agent != a {
			a.peers = append(a.peers, agent)
		}
	}
	a.shares = make([]int, numberOfAgents)
}

func (a Agent) run(wg *sync.WaitGroup) {

	defer wg.Done()

	//get coeffs for polynomial
	var polynomialCoefficients []int
	for i := 0; i < t-1; i++ {
		negate := rand.Int()&1 > 0
		randomCoefficient := rand.Intn(q-1) + 1
		if negate {
			randomCoefficient = -randomCoefficient
		}
		polynomialCoefficients = append(polynomialCoefficients, randomCoefficient)
	}

	//the secret to share is the constant term of the polynomial
	// secret := polynomialCoefficients[0]

	// using Z mod q under addition as our cyclic group, the generator is 1
	// in this case, the commitments are the same as the polynomial coefficients
	var commitments []int
	for _, coeff := range polynomialCoefficients {
		commitments = append(commitments, coeff%q)
	}

	var secretShares []int
	for i := 1; i <= numberOfAgents; i++ {
		secretShares = append(secretShares, evaluatePolynomial(polynomialCoefficients, i))
	}

	for _, agent := range a.peers {
		agent.tell(Message{
			Type: SecretShare,
			From: a.index,
			Value: secretShares[agent.index],
		})
	}

	for {
		select {
		case msg := <-a.inbox:
			a.handleMessage(msg)
		default:
			println("Done")
			return
		}
	}

}

func (a *Agent) handleMessage(message Message) {
	switch message.Type {
	case SecretShare:
		a.shares[message.From] = message.Value
	}
}

func (a Agent) tell(message Message) {
	a.inbox <- message
}

func main() {
	var agents []*Agent
	var wg sync.WaitGroup

	for i := 0; i < numberOfAgents; i++ {
		agents = append(agents, newAgent(i))
	}
	for _, agent := range agents {
		agent.providePeers(agents)
	}

	for _, agent := range agents {
		wg.Add(1)
		go agent.run(&wg)
	}

	wg.Wait()
}
