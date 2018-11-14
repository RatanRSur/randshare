package main

import (
	"fmt"
	"math/rand"
	"sync"
)

const numberOfAgents = 10
const t = 3
const q int64 = 17

var agents []*Agent

type ZmodQ struct {
	q int64
	G int64
}

func (g ZmodQ) Times(x int64, y int64) int64 { return mod(x+y, g.q) }
func (g ZmodQ) Exp(x, y int64) int64         { return mod(mod(x, g.q)*mod(y, g.q), g.q) }
func (g ZmodQ) Identity() int64              { return 0 }
func (g ZmodQ) Inverse(n int64) int64        { return g.q - n }

var zmodq ZmodQ = ZmodQ{q, 1}

type Group interface {
	Add(int64) int64
	Identity() int64
	Inverse(int64) int64
}

type Agent struct {
	inbox                  chan Message
	index                  int64
	peers                  []*Agent
	polynomialCoefficients [t - 1]int64
	commitments            [numberOfAgents+1][t - 1]int64
	shares                 []int64
	validSharesReceived    [numberOfAgents+1]bool
}

type MessageType uint8

const (
	SecretShare MessageType = iota
	Commitment
)

type Message struct {
	Type          MessageType
	From          int64
	IntValue      int64
	IntArrayValue []int64
}

func mod(d, m int64) int64 {
	var res int64 = d % m
	if (res < 0 && m > 0) || (res > 0 && m < 0) {
		return res + m
	}
	return res
}

func pow(x int64, y int64) int64 {
	result := int64(1)
	for i := int64(0); i < y; i++ {
		result *= x
	}
	return result
}

func evaluatePolynomial(coeffs [t - 1]int64, x int64) int64 {
	result := int64(0)
	for k, coeff := range coeffs {
		result += coeff * pow(x, int64(k))
	}
	return result
}

func newAgent(index int64) *Agent {
	inbox := make(chan Message, numberOfAgents*numberOfAgents)
	return &Agent{inbox: inbox, index: index+1}
}

func (a *Agent) providePeers(agents []*Agent) {
	for _, agent := range agents {
		if agent != a {
			a.peers = append(a.peers, agent)
		}
	}
	a.shares = make([]int64, numberOfAgents+1)
}

func (a Agent) run(wg *sync.WaitGroup) {

	defer wg.Done()

	//get coeffs for polynomial
	for k := 0; k < t-1; k++ {
		negate := rand.Int() & 1 > 0
		randomCoefficient := rand.Int63n(q-1) + 1
		if negate {
			randomCoefficient = -randomCoefficient
		}
		a.polynomialCoefficients[k] = randomCoefficient
	}

	//the secret to share is the constant term of the polynomial
	// secret := a.polynomialCoefficients[0]

	// TODO: use better group later
	var commitments [t - 1]int64
	for k, coeff := range a.polynomialCoefficients {
		commitments[k] = zmodq.Exp(zmodq.G, coeff)
	}

	var secretShares []int64
	for i := 1; i <= numberOfAgents+1; i++ {
		secretShares = append(secretShares,
			evaluatePolynomial(a.polynomialCoefficients, int64(i)))
	}

	// broadcast commitments
	broadcast(Message{
		Type:          Commitment,
		From:          a.index,
		IntArrayValue: commitments[:],
	})

	for _, agent := range a.peers {
		agent.tell(Message{
			Type:     SecretShare,
			From:     a.index,
			IntValue: secretShares[agent.index],
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
		a.shares[message.From] = message.IntValue
		// i = a.index
		// x = a.index
		// j = message.From
		// s_j(i) = message.IntValue
		// k = k
		// G
		// println("Got SecretSharefrom: ", message.From)
		accumulator := zmodq.Identity()
		var str string
		for k, commitment := range a.commitments[message.From] {
			accumulator = zmodq.Times(accumulator,
				zmodq.Exp(commitment, zmodq.Exp(a.index, int64(k))))
			str += fmt.Sprintf("k: %d, i: %d, com: %d, acc: %d\n", k, a.index, commitment, accumulator)
		}
		println(str)
		verificationTarget := zmodq.Exp(zmodq.G, message.IntValue)
		if accumulator != verificationTarget {
			println("acc: ", accumulator)
			println("target: ", verificationTarget)
			println("NOOOOOOOOOOOOO!")
		} else {
			println("OK!!")
		}
	case Commitment:
		// println("Got commitment from: ", message.From)
		copy(a.commitments[message.From][:], message.IntArrayValue)
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
		agents = append(agents, newAgent(int64(i)))
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
