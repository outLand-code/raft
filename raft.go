package raft

type Raft struct {
	//RPC server to send message
	server *Server

	id int

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
}

func (r *Raft) RequestVote(args VoteArgs, reply *VoteReply) error {
	return nil
}

func (r *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	return nil
}
