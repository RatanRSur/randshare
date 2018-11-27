package main

import (
	"fmt"
	"math/rand"
	"sync"
)

const numberOfAgents = 10
const t = 4
const f = t - 1
const q int64 = 7

var agents []*Agent

type ZmodQ struct {
	q int64
	G int64
}

func (g ZmodQ) Times(x int64, y int64) int64 { return (x + y) % g.q }
func (g ZmodQ) Exp(x, y int64) int64 {
	accumulator := g.Identity()
	for i := int64(0); i < y; i++ {
		accumulator = g.Times(accumulator, x)
	}
	return accumulator
}
func (g ZmodQ) Identity() int64       { return 0 }
func (g ZmodQ) Inverse(n int64) int64 { return g.q - n }

var zmodq ZmodQ = ZmodQ{q, 1}

type Group interface {
	Times(int64, int64) int64
	Identity() int64
	Inverse(int64) int64
}

type Agent struct {
	inbox                   chan Message
	index                   int64
	peers                   []*Agent
	polynomialCoefficients  [t - 1]int64
	commitments             [numberOfAgents + 1][t - 1]int64
	shares                  [numberOfAgents + 1]int64
	validShares             [numberOfAgents + 1]int64
	positiveShareVotes      [numberOfAgents + 1]int64
	negativeShareVotes      [numberOfAgents + 1]int64
	positiveCommitmentVotes [numberOfAgents + 1]int64
	negativeCommitmentVotes [numberOfAgents + 1]int64
	finalSharesReceived     int
}

type MessageType uint8

const (
	SecretShare MessageType = iota
	Commitment
	ShareVote
	CommitmentVote
	FinalValidShareSrslyGuys
)

type Message struct {
	Type          MessageType
	From          int64
	IntValue      int64
	IntArrayValue []int64
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
	a := &Agent{inbox: inbox, index: index + 1}
	for i := range a.validShares {
		a.validShares[i] = -1
	}
	return a
}

func (a *Agent) providePeers(agents []*Agent) {
	for _, agent := range agents {
		if agent != a {
			a.peers = append(a.peers, agent)
		}
	}
}

func (a Agent) run(wg *sync.WaitGroup) {

	defer wg.Done()

	//get coeffs for polynomial
	for k := 0; k < t-1; k++ {
		a.polynomialCoefficients[k] = rand.Int63n(q-1) + 1
	}

	//the secret to share is the constant term of the polynomial
	// secret := a.polynomialCoefficients[0]

	// TODO: use better group later
	var commitments [t - 1]int64
	for k, coeff := range a.polynomialCoefficients {
		commitments[k] = zmodq.Exp(zmodq.G, coeff)
	}

	var secretShares [numberOfAgents + 1]int64
	for j := 1; j <= numberOfAgents; j++ {
		secretShares[j] = evaluatePolynomial(a.polynomialCoefficients, int64(j))
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
			return
		}
	}
}

func (a *Agent) handleMessage(message Message) {
	switch message.Type {
	case SecretShare:
		a.shares[message.From] = message.IntValue
		verificationTarget := zmodq.Exp(zmodq.G, message.IntValue)
		accumulator := zmodq.Identity()
		for k, commitment := range a.commitments[message.From] {
			nextTerm := zmodq.Exp(commitment, pow(a.index, int64(k)))
			accumulator = zmodq.Times(accumulator, nextTerm)
		}

		var votePayload int64
		if accumulator == verificationTarget {
			votePayload = 1
		} else {
			votePayload = message.IntValue
		}
		broadcast(Message{
			Type:          ShareVote,
			From:          a.index,
			IntArrayValue: []int64{a.index, message.From, votePayload},
		})
	case Commitment:
		copy(a.commitments[message.From][:], message.IntArrayValue)
	case ShareVote:
		voteSubject := message.IntArrayValue[1]
		if message.IntArrayValue[2] == 1 {
			a.positiveShareVotes[voteSubject] += 1
			if a.positiveShareVotes[voteSubject] == 2*f+1 {
				votePayload := int64(1)
				broadcast(Message{
					Type:          CommitmentVote,
					From:          a.index,
					IntArrayValue: []int64{a.index, message.From, votePayload},
				})
			}
		} else {
			a.negativeShareVotes[voteSubject] += 1
			if a.negativeShareVotes[voteSubject] == f+1 {
				votePayload := int64(0)
				broadcast(Message{
					Type:          CommitmentVote,
					From:          a.index,
					IntArrayValue: []int64{a.index, message.From, votePayload},
				})
			}
		}
	case CommitmentVote:
		voteSubject := message.IntArrayValue[1]
		vote := message.IntArrayValue[2]
		if vote == 1 {
			a.positiveCommitmentVotes[voteSubject] += 1
			if a.positiveCommitmentVotes[voteSubject] == 2*f+1 {
				a.validShares[voteSubject] = vote
			}
		} else {
			a.negativeCommitmentVotes[voteSubject] += 1
			if a.negativeCommitmentVotes[voteSubject] == 2*f+1 {
				a.validShares[voteSubject] = vote
			}
		}
		allDecided := true
		var validatedShareParties []int
		for j, valid := range a.validShares {
			if valid == -1 {
				allDecided = false
				break
			}
			validatedShareParties = append(validatedShareParties, j)
		}
		if allDecided {
			if len(validatedShareParties) > f {
				for _, j := range validatedShareParties {
					broadcast(Message{
						Type:     FinalValidShareSrslyGuys,
						From:     a.index,
						IntValue: a.shares[j],
					})
				}
			}
		}
	case FinalValidShareSrslyGuys:
		a.finalSharesReceived += 1
		if a.finalSharesReceived == t {
			// do lagrange interpolation here to roc

		}
	}
}

type Point struct{ x, y int64 }

func lagrangeInterpolation(x int64, points []Point) int64 {
	agg := int64(0)
	for j, point := range points {
		agg += point.y * l(x, j, points)
	}
	return agg
}

func l(x int64, j int, points []Point) int64 {
	agg := int64(1)
	for i, point := range points {
		if i == j {
			continue
		}
		agg *= (x - point.x) / (points[j].x - point.x)
	}
	return agg
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
	fmt.Printf("")
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
