package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type Server struct {
	rpcServer *rpc.Server
	listener  net.Listener
	rcvr      *Rcvr

	rpcClients []*rpc.Client

	exit chan interface{}

	mu sync.Mutex
}

//Rcvr define
type Rcvr interface {
	//RequestVote invoked by candidates to gather votes
	RequestVote(args VoteArgs, reply *VoteReply) error
	//AppendEntries invoked by leader to replicate log entries ,also used as heartbeat
	AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error
	//InstallSnapshot invoked by leader to send chunks of snapshot to a follower.leader always send chunks in order.
	InstallSnapshot(args InstallSnapshotArgs, reply *InstallSnapshotReply) error
	//SetNewLogEntry invoked by client to send a command to server
	SetNewLogEntry(args Command, reply *ClientResp) error
}

//NewServer create a new RPC server
func NewServer(rcvr *Rcvr) *Server {
	return &Server{
		rcvr: rcvr,
		exit: make(chan interface{}),
	}
}

// Start
func (s *Server) Start() {
	serv := rpc.NewServer()
	_ = serv.RegisterName(RPCRegisterName, *s.rcvr)
	s.rpcServer = serv

	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", Config.Port))
	if err != nil {
		panic(err)
	}

	go func() {
		tick := time.NewTicker(Config.rpcClientCheckTime)
		for {
			s.clients()
			<-tick.C
		}
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.exit:
				return
			default:
				log.Printf("accept error:%v ,from %s\n", err, conn.RemoteAddr())
			}
			continue
		}
		go func() {
			s.rpcServer.ServeConn(conn)
		}()
	}
}

func (s *Server) Exit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.rpcClients {
		if s.rpcClients[id] != nil {
			_ = s.rpcClients[id].Close()
			s.rpcClients[id] = nil
		}
	}
	close(s.exit)
	_ = s.listener.Close()
}

func (s *Server) clients() {
	//log.Printf("rpc client check %v\n", s.rpcClients)
	if s.rpcClients == nil {
		s.rpcClients = make([]*rpc.Client, len(Config.Cluster))
	}
	for index, addr := range Config.Cluster {
		if s.rpcClients[index] == nil {
			c, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("rpc client create error from %s,error:%v\n", addr, err)
				continue
			}
			s.rpcClients[index] = c
		}
	}
}

func (s *Server) getListenAddr() string {
	return s.listener.Addr().String()
}
