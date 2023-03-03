package raft

const (
	EnvKeyRaftUseLocalConfig        = "env_key_raft_use_local_config"
	ElectionBaseTimeOut      int64  = 150
	RPCRegisterName                 = "RaftSpace"
	RFollower                Rstate = iota
	RCandidate
	RLeader
)

type Rstate int

type VoteArgs struct {
	term         int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}
type VoteReply struct {
	term        int
	voteGranted bool
}

type AppendEntriesArgs struct {
	term         int
	leaderId     int
	prevLogIndex int
	prevLogTerm  int
	leaderCommit int
	entries      []LogEntry
}
type AppendEntriesReply struct {
	term    int
	success bool
}

type LogEntry struct {
	term    int
	command interface{}
}
