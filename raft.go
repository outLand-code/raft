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
	initLen := len(Config.Cluster)
	return &Raft{
		id:            id,
		state:         RFollower,
		currentTerm:   0,
		votedFor:      -1,
		prevElectTime: time.Now(),
		commitIndex:   -1,
		lastApplied:   -1,
		nextIndex:     make([]int, initLen),
		matchIndex:    InitIntArray(initLen, -1),
		log:           []LogEntry{},
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
			lastIndex, lastTerm := r.getLastLogIndexAndTerm()
			args := VoteArgs{
				Term:         term,
				CandidateId:  r.id,
				LastLogIndex: lastIndex,
				LastLogTerm:  lastTerm,
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
				nextIndex := r.nextIndex[id]
				prevIndex := nextIndex - 1
				prevTerm := 0
				if prevIndex > -1 {
					prevTerm = r.log[prevIndex].Term
				}

				heartbeat := AppendEntriesArgs{
					Term:         r.currentTerm,
					LeaderId:     r.id,
					PrevLogIndex: prevIndex,
					PrevLogTerm:  prevTerm,
					LeaderCommit: r.commitIndex,
					Entries:      r.log[nextIndex:],
				}
				reply := &AppendEntriesReply{Success: false}
				if err := r.server.rpcClients[id].Call(RPCRegisterName+".AppendEntries", heartbeat, reply); err == nil {
					log.Printf("peer ID:%d,AppendEntries reply %v\n", id, reply)
					log.Printf("peer ID:%d,AppendEntries args %v\n", id, heartbeat)
					if reply.Term > r.currentTerm {
						r.toBeFollower(reply.Term)
						return
					}
					if reply.Success {

						r.nextIndex[id] += len(heartbeat.Entries)
						r.matchIndex[id] = r.nextIndex[id] - 1
						matchCount := 0
						for _, mi := range r.matchIndex {
							if mi > nextIndex {
								matchCount++
							}
						}
						if matchCount*2 >= len(r.matchIndex) {
							r.commitIndex += 1
						}

					} else {
						log.Printf("peer ID:%d reply false ,the trem and index of the log is mismatching\n", id)
						r.nextIndex[id] = nextIndex - 1
					}

				} else {
					log.Printf("peer ID:%d send heartbeat may be fail,error:%v\n", id, err)
				}

			}(id)
		}
		<-tick.C
	}

}

func (r *Raft) commit() {

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

	reply.VoteGranted = false
	if args.Term > r.currentTerm {
		r.toBeFollower(args.Term)
	}

	//only Args' term equals the Raft's current term is meaningful,other situations will reply false

	//if args.Term < r.currentTerm {
	//	reply.VoteGranted = false
	//}

	lastIndex, lastTerm := r.getLastLogIndexAndTerm()
	if args.Term == r.currentTerm && (r.votedFor == -1 || r.votedFor == args.CandidateId) &&
		args.LastLogTerm == lastTerm && args.LastLogIndex == lastIndex {
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

	log.Printf("AppendEntries args is %v,current Raft :%v\n", args, r)

	r.mu.Lock()
	defer r.mu.Unlock()

	reply.Success = false
	if args.Term > r.currentTerm {
		r.toBeFollower(args.Term)
	} else if args.Term == r.currentTerm {
		if r.state != RFollower {
			r.toBeFollower(args.Term)
		}
		reply.Success = true
	}

	if args.PrevLogTerm == r.currentTerm && args.PrevLogIndex > len(r.log) {
		reply.Success = false
	} else {
		if args.PrevLogIndex > -1 {
			prevLog := r.log[args.PrevLogIndex]
			if prevLog.Term != args.PrevLogTerm {
				r.log = r.log[0 : args.PrevLogIndex-1]
				reply.Success = false
			}
		}
	}

	if reply.Success {
		r.votedFor = args.LeaderId
		r.prevElectTime = time.Now()
		if args.Entries != nil && len(args.Entries) > 0 {
			r.log = append(r.log, args.Entries...)
		}

		if args.LeaderCommit > r.commitIndex {
			r.commitIndex = Min(args.LeaderCommit, len(r.log)-1)
		}
	}
	log.Printf("the Raft log:%v\n", r.log)
	reply.Term = r.currentTerm
	return nil
}

func (r *Raft) InstallSnapshot(args InstallSnapshotArgs, reply *InstallSnapshotReply) error {

	return nil
}

func (r *Raft) SetNewLogEntry(args Command, reply *ClientResp) error {
	if r.state == RLeader {
		args.Term = r.currentTerm
		r.log = append(r.log, LogEntry{Term: r.currentTerm, Command: args})
		reply.Success = true
	} else {
		reply.LeaderAddress = Config.Cluster[r.votedFor]
		reply.Success = false
	}

	return nil
}

func (r *Raft) getLastLogIndexAndTerm() (index int, term int) {
	l := len(r.log)
	if l == 0 {
		index = -1
		term = -1
	} else {
		index = l - 1
		term = r.log[l-1].Term
	}
	return
}
