package main

import (
	_ "fmt"
	"math/rand"
	"sync"
)

const numberOfAgents = 10
const t = 4
const q int = 2166136261

var agents []*Agent

type ZmodQ struct {
	q int
	G int
}

func (g ZmodQ) Times(x int, y int) int   { return (x + y) % g.q }
func (g ZmodQ) Exp(x, y int) int  { return x * y }
func (g ZmodQ) Identity() int     { return 0 }
func (g ZmodQ) Inverse(n int) int { return g.q - n }

var zmodq ZmodQ = ZmodQ{q, 1}

type Group interface {
	Add(int) int
	Identity() int
	Inverse(int) int
}

type Agent struct {
	inbox                  chan Message
	index                  int
	peers                  []*Agent
	polynomialCoefficients [t - 1]int
	commitments            [t - 1]int
	shares                 []int
	validSharesReceived    [numberOfAgents]bool
}

type MessageType uint8

const (
	SecretShare MessageType = iota
	Commitment
)

type Message struct {
	Type          MessageType
	From          int
	IntValue      int
	IntArrayValue []int
}

func pow(x int, y int) int {
	result := 1
	for i := 0; i < y; i++ {
		result *= x
	}
	return result
}

func evaluatePolynomial(coeffs [t - 1]int, x int) int {
	result := 0
	for i, coeff := range coeffs {
		result += coeff * pow(x, i)
	}
	return result
}

func newAgent(index int) *Agent {
	inbox := make(chan Message, numberOfAgents*numberOfAgents)
	return &Agent{inbox: inbox, index: index}
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
	for i := 0; i < t-1; i++ {
		negate := rand.Int()&1 > 0
		randomCoefficient := rand.Intn(q-1) + 1
		if negate {
			randomCoefficient = -randomCoefficient
		}
		a.polynomialCoefficients[i] = randomCoefficient
	}

	//the secret to share is the constant term of the polynomial
	// secret := a.polynomialCoefficients[0]

	// TODO: use better group later
	// using Z mod q under addition as our cyclic group, the generator is 1
	// in this case, the commitments are the same as the polynomial coefficients
	for k, coeff := range a.polynomialCoefficients {
		a.commitments[k] = zmodq.Exp(zmodq.G, coeff)
	}

	var secretShares []int
	for i := 1; i <= numberOfAgents; i++ {
		secretShares = append(secretShares, evaluatePolynomial(a.polynomialCoefficients, i))
	}

	for _, agent := range a.peers {
		agent.tell(Message{
			Type:     SecretShare,
			From:     a.index,
			IntValue: secretShares[agent.index],
		})
	}

	// broadcast commitments
	broadcast(Message{
		Type:          Commitment,
		From:          a.index,
		IntArrayValue: a.commitments[:],
	})

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
		a.shares[message.From] = message.IntValue
		println("Got SecretShare")
                accumulator := zmodq.Identity()
                for k, commitment := range a.commitments {
                    accumulator = zmodq.Times(accumulator, zmodq.Exp(commitment, zmodq.Exp(message.From, k)))
                }
                verificationTarget := zmodq.Exp(zmodq.G, evaluatePolynomial(a.polynomialCoefficients, message.From))
                if accumulator != verificationTarget {
                    println(accumulator)
                    println(verificationTarget)
                    panic("NOOOOOOOOOOOOO!")
                }
	case Commitment:
		println("Got Commitment")
	}
}

func (a Agent) tell(message Message) {
	a.inbox <- message
}

func broadcast(message Message) {
	for _, agent := range agents {
		if agent.index != message.From {
			agent.tell(message)
		}
	}
}

func main() {
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
