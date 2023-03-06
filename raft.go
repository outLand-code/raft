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

	mu sync.Mutex
}

func NewRaft(id int) *Raft {
	if Config == nil {
		if err := loadConfig(); err != nil {
			log.Fatalf("load config error:%v\n", err)
		}
	}
	return &Raft{
		id:            id,
		state:         RFollower,
		currentTerm:   0,
		votedFor:      -1,
		prevElectTime: time.Now(),
	}
}

func NewRaftWithConfig(config *RConfig) *Raft {
	if config == nil {
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
	log.Printf("start election timer, current term:%d , timeout value:%d ms\n", r.currentTerm, electionTimeout/1000000)
	tick := time.NewTicker(10 * time.Millisecond)

	for {
		<-tick.C
		if r.state != RFollower && r.state != RCandidate {
			log.Printf("election timer end ,raft current state is %s\n", transStateStr(r.state))
			return
		}

		if time.Since(r.prevElectTime) > electionTimeout {
			go r.election()
		}
	}

}

func (r *Raft) election() {

	r.state = RCandidate
	r.currentTerm += 1
	term := r.currentTerm
	r.prevElectTime = time.Now()
	r.votedFor = r.id

	log.Printf("election the Raft state is %s, start leader election with term:%d\n", transStateStr(r.state), term)
	voteCount := 0
	for id := range r.server.rpcClients {
		if r.server.rpcClients[id] == nil {
			continue
		}
		go func(id int) {

			args := VoteArgs{
				Term:         term,
				CandidateId:  r.id,
				LastLogIndex: len(r.log) - 1,
				LastLogTerm:  r.getLastLogTerm(),
			}
			reply := &VoteReply{VoteGranted: false}
			err := r.server.rpcClients[id].Call(RPCRegisterName+".RequestVote", args, reply)
			if err != nil {
				log.Printf("RPC request is wrong ,error :%v\n", err)
				return
			}
			log.Printf("peer ID:%d,RequestVote reply %v\n", id, reply)

			if r.state != RCandidate {
				return
			}
			if reply.Term > r.currentTerm {
				r.toBeFollower(reply.Term)
				return
			} else if reply.Term == r.currentTerm {
				if reply.VoteGranted {

					if voteCount += 1; voteCount*2 >= len(r.server.rpcClients) {
						r.toBeLeader()
					}
				}
			}

		}(id)
	}

}

//getElectionTimeOut return election time out, this value is between 150ms and 300ms in the Raft page
func getElectionTimeOut() time.Duration {
	rand.Seed(time.Now().Unix())
	return time.Duration(rand.Intn(Config.electionTimeout)+Config.electionTimeout) * time.Millisecond
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

	r.currentTerm = term
	r.state = RFollower
	r.votedFor = -1
	r.prevElectTime = time.Now()
	go r.electionTimer()
	log.Printf("toBeFollower the Raft state is %s\n", transStateStr(r.state))

}

func (r *Raft) toBeLeader() {

	r.state = RLeader
	tick := time.NewTicker(Config.heartBeatInterval)
	log.Printf("toBeLeader the Raft state is %s and begin to start send heartbeat\n", transStateStr(r.state))
	for {

		for id := range r.server.rpcClients {
			if r.server.rpcClients[id] == nil {
				continue
			}
			go func(id int) {
				heartbeat := AppendEntriesArgs{
					Term:         r.currentTerm,
					LeaderId:     r.id,
					PrevLogIndex: -1,
					PrevLogTerm:  -1,
					LeaderCommit: -1,
					Entries:      nil,
				}
				reply := &AppendEntriesReply{Success: false}
				if err := r.server.rpcClients[id].Call(RPCRegisterName+".AppendEntries", heartbeat, reply); err == nil {
					log.Printf("peer ID:%d,AppendEntries reply %v\n", id, reply)
					if reply.Term > r.currentTerm {
						r.toBeFollower(reply.Term)
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

	r.mu.Lock()
	defer r.mu.Unlock()

	if args.Term > r.currentTerm {
		r.toBeFollower(args.Term)
	}
	if args.Term < r.currentTerm {
		reply.VoteGranted = false
	} else if args.Term == r.currentTerm && (r.votedFor == -1 || r.votedFor == args.CandidateId) {
		reply.VoteGranted = true
		r.votedFor = args.CandidateId
		r.prevElectTime = time.Now()
	}
	reply.Term = r.currentTerm
	return nil
}

//AppendEntries the receiver should implement below:
//if Raft's current term  greater than Args's term ,set voteGranted value false;
//Finally , update the value of Raft's prevElectTime to current time if reply true
func (r *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {

	//log.Printf("AppendEntries args is %v,current Raft :%v\n", args, r)

	r.mu.Lock()
	defer r.mu.Unlock()

	if args.Term > r.currentTerm {
		r.toBeFollower(args.Term)
	} else if args.Term == r.currentTerm {
		if r.state != RFollower {
			r.toBeFollower(args.Term)
		}
		reply.Success = true
	}

	if reply.Success {
		r.prevElectTime = time.Now()
	}
	reply.Term = r.currentTerm
	return nil
}

func (r *Raft) getLastLogTerm() int {
	if len(r.log) == 0 {
		return -1
	}
	return r.log[len(r.log)-1].Term
}
