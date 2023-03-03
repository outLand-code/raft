package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

type Server struct {
	rpcServer *rpc.Server
	listener  net.Listener
	rcvr      *Rcvr

	rpcClients map[int]*rpc.Client

	exit chan interface{}

	mu sync.Mutex
}

type Rcvr interface {
	//RequestVote invoked by candidates to gather votes
	RequestVote(args VoteArgs, reply *VoteReply) error
	//AppendEntries invoked by leader to replicate log entries ,also used as heartbeat
	AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error
}

func NewServer(rcvr *Rcvr) *Server {
	return &Server{
		rcvr: rcvr,
		exit: make(chan interface{}),
	}
}

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
	}()
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
