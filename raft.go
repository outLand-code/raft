package raft

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type Raft struct {
	//RPC server to send message
	server *Server

	id    int
	state Rstate

	//volatile state on all servers
	commitIndex int
	lastApplied int

	//volatile state on leaders
	nextIndex  []int
	matchIndex []int

	//persistent state on all servers
	currentTerm int
	votedFor    int
	log         []LogEntry

	//store previous time of election leader
	prevElectTime time.Time
	//store the IDs of other Raft's instances
	peerIds []int

	mu sync.Mutex
}

func NewRaft(id int) *Raft {
	if Config == nil {
		if err := loadConfig(); err != nil {
			log.Fatalf("load config error:%v\n", err)
		}
	}
	return &Raft{
		id:          id,
		state:       RFollower,
		currentTerm: 0,
		votedFor:    -1,
	}
}

func NewRaftWithConfig(config *RConfig) *Raft {
	if config != nil {
		log.Fatal("raft config is null,please set the config\n")
	}
	Config = NewConfig(config)
	return NewRaft(Config.Id)
}

func (r *Raft) Run() {
	var rcvr Rcvr = r
	r.server = NewServer(&rcvr)
	go r.electionTimer()
	r.server.Start()
}

func (r *Raft) electionTimer() {
	electionTimeout := getElectionTimeOut()
	log.Printf("start election timer, current term:%d , timeout value:%d ms\n", electionTimeout, r.currentTerm)
	tick := time.NewTicker(10 * time.Millisecond)

	for {
		<-tick.C
		if r.state != RFollower && r.state != RCandidate {
			log.Printf("election timer end ,raft current state is %s\n", transStateStr(r.state))
			return
		}
		if time.Since(r.prevElectTime) > time.Duration(electionTimeout) {
			go r.election()
		}
	}

}

func (r *Raft) election() {

	r.state = RCandidate
	r.mu.Lock()
	term := r.currentTerm
	term += 1
	r.currentTerm = term
	r.prevElectTime = time.Now()
	r.mu.Unlock()

	log.Printf("election the Raft state is %s, start leader election with term:%d\n", transStateStr(r.state), term)
	voteCount := 0
	for id, _ := range r.peerIds {

		go func(id int) {

			args := VoteArgs{
				term:         term,
				candidateId:  r.id,
				lastLogIndex: len(r.log) - 1,
				lastLogTerm:  r.log[len(r.log)-1].term,
			}
			reply := &VoteReply{voteGranted: false}
			err := r.server.rpcClients[id].Call(RPCRegisterName+".RequestVote", args, reply)
			if err != nil {
				log.Printf("RPC request is wrong ,error :%v\n", err)
				return
			}
			if reply.term > r.currentTerm {
				r.toBeFollower(reply.term)
				return
			}
			if reply.voteGranted {

				if voteCount += 1; voteCount*2 >= len(r.peerIds) {
					r.toBeLeader()
				}
			}

		}(id)
	}

}

//getElectionTimeOut return election time out, this value is between 150ms and 300ms in the Raft page
func getElectionTimeOut() int64 {
	rand.Seed(time.Now().Unix())
	return rand.Int63n(ElectionBaseTimeOut) + ElectionBaseTimeOut
}

func transStateStr(state Rstate) (strState string) {
	switch state {
	case RFollower:
		strState = "Follower"
	case RCandidate:
		strState = "Candidate"
	case RLeader:
		strState = "Leader"
	default:
		strState = "Unknown"
	}
	return
}

func (r *Raft) toBeFollower(term int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == RFollower {
		return
	}
	r.currentTerm = term
	r.state = RFollower
	go r.electionTimer()
	log.Printf("toBeFollower the Raft state is %s\n", transStateStr(r.state))

}

func (r *Raft) toBeLeader() {

	r.state = RLeader
	tick := time.NewTicker(10 * time.Millisecond)
	log.Printf("toBeLeader the Raft state is %s and begin to start send heartbeat\n", transStateStr(r.state))
	for {

		for id := range r.peerIds {
			go func(id int) {
				heartbeat := AppendEntriesArgs{
					term:         r.currentTerm,
					leaderId:     r.id,
					prevLogIndex: -1,
					prevLogTerm:  -1,
					leaderCommit: -1,
					entries:      nil,
				}
				reply := &AppendEntriesReply{success: false}
				if err := r.server.rpcClients[id].Call(RPCRegisterName+".AppendEntries", heartbeat, reply); err == nil {

					if reply.term > r.currentTerm {
						r.toBeFollower(reply.term)
						return
					}

				} else {
					log.Printf("peer ID:%d send heartbeat may be fail,error:%v\n", id, err)
				}

			}(id)
		}
		<-tick.C
	}

}

// RequestVote the receiver should implement below:
// if Raft's current term  greater than Args's term ,set voteGranted value false;
// if Args's voteFot is null or candidateId,and candidate's log should be at least as up-to-data as receiver's log,
// which means that the length of Raft's logs should equal to Args's lastLogIndex and the last term in Raft's logs equal
// to Args's lastLogTerm;
func (r *Raft) RequestVote(args VoteArgs, reply *VoteReply) error {
	log.Printf("RequestVote args is %v,current Raft :%v\n", args, r)
	if args.term < r.currentTerm {
		reply.voteGranted = false
	} else if (r.votedFor == -1 || r.votedFor == args.candidateId) &&
		args.lastLogIndex == len(r.log) && args.lastLogTerm == r.log[len(r.log)-1].term {
		reply.voteGranted = true
	}
	reply.term = r.currentTerm
	return nil
}

//AppendEntries the receiver should implement below:
//if Raft's current term  greater than Args's term ,set voteGranted value false;
//Finally , update the value of Raft's prevElectTime to current time if reply true
func (r *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.Printf("AppendEntries args is %v,current Raft :%v\n", args, r)

	reply.term = r.currentTerm
	reply.success = true
	if args.term < r.currentTerm {
		reply.success = false
	}

	if reply.success {
		r.prevElectTime = time.Now()
	}

	return nil
}
