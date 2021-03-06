package twophasecommit

import (
	"fmt"
	"time"
)

// CoordinatorHandler represents a handler to handle the messages for the coordinator.
type CoordinatorHandler struct {
	canCommitChannel chan<- CanCommit // channel for sending CanCommit messages
	voteChannel      chan Vote        // channel for receiving Vote messages
	decisionChannel  chan<- Decision  // channel for sending Decision messages
	ackChannel       chan Ack         // channel for receiving Ack messages
}

// NewCoordinator returns a coordinator handler for a new coordinator. It takes the
// following arguments:
//
// canCommitChannel: a send only channel used to send cancommit message to the test adapter/workers.
//
// decisionChannel: a send only channel used to send decision message to the test adapter/workers.
func NewCoordinator(canCommitChannel chan<- CanCommit, decisionChannel chan<- Decision) *CoordinatorHandler {

	return &CoordinatorHandler{
		canCommitChannel: canCommitChannel,
		voteChannel:      make(chan Vote, 16),
		decisionChannel:  decisionChannel,
		ackChannel:       make(chan Ack, 16),
	}
}

// Start starts c's main run loop as a separate goroutine. The main run loop
// sends cancommit and decision messages to the test adapter/workers, and handles incoming
// vote messages and ack messages.
func (c *CoordinatorHandler) Start(numOfWorker int, finalDecisionChannel chan DecisionEnum) {

	// send canCommit messages
	for w := 1; w <= numOfWorker; w++ {
		canCommit := CanCommit{WorkerID(w)}
		c.SendCanCommit(canCommit)
	}

	var votes []Vote
	numOfCommit := 0

	// start goroutine
	go func() {
		fd := Abort
		for {
			select {
			case v, _ := <-c.voteChannel:
				fmt.Println("Received vote from worker:", v.String())
				if len(votes) < numOfWorker {
					votes = c.CollectVotes(v, votes)
					if len(votes) == numOfWorker {
						for _, v := range votes {
							switch v.GetVote() {
							case Yes:
								numOfCommit++
							case No:
								abort := Decision{v.GetWorkID(), Abort}
								c.SendDecision(abort)
								fd = Abort
							}
						}
						if numOfCommit == numOfWorker {
							for _, v := range votes {
								commit := Decision{v.GetWorkID(), Commit}
								c.SendDecision(commit)
							}
							fd = Commit
						}
						c.SendFinalDecision(finalDecisionChannel, fd)
					}
				}
			case a, _ := <-c.ackChannel:
				fmt.Println("Received ack from works:", a.String())
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
}

// SendFinalDecision sends the final decision of the coordinator to the test adapter/workers.
func (c *CoordinatorHandler) SendFinalDecision(fdchannel chan DecisionEnum, fd DecisionEnum) {
	fdchannel <- fd
}

// SendCanCommit sends cancommit message to the test adapter/workers.
func (c *CoordinatorHandler) SendCanCommit(cc CanCommit) {
	c.canCommitChannel <- cc
}

// SendDecision sends decision message to the test adapter/workers.
func (c *CoordinatorHandler) SendDecision(d Decision) {
	c.decisionChannel <- d
}

// DeliverVote receives vote message from the test adapter/workers.
func (c *CoordinatorHandler) DeliverVote(v Vote) {
	c.voteChannel <- v
}

// DeliverACK receives ack message from the test adapter/workers.
func (c *CoordinatorHandler) DeliverACK(a Ack) {
	c.ackChannel <- a
}

// CollectVotes collect received votes.
func (c *CoordinatorHandler) CollectVotes(v Vote, votes []Vote) []Vote {
	return append(votes, v)
}
