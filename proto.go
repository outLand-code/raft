package raft

const (
	RPCRegisterName        = "RaftSpace"
	RFollower       Rstate = iota
	RCandidate
	RLeader
)

type Rstate int

type VoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}
type VoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	LeaderCommit int
	Entries      []LogEntry
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

type LogEntry struct {
	Term    int
	Command Command
}

type Command struct {
	Term int
	KeyValuePair
}

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Offset            int
	Data              []byte
	Done              bool
}

type InstallSnapshotReply struct {
	Term int
}

type KeyValuePair struct {
	Key   string
	Value interface{}
}

type ClientResp struct {
	Success       bool
	LeaderAddress string
}
