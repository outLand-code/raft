package raft

import (
	"log"
	"net/rpc"
	"testing"
	"time"
)

var times = 500

func TestRaft1(t *testing.T) {
	raft := NewRaftWithConfig(&RConfig{
		Id:      1,
		Port:    9101,
		Cluster: []string{"127.0.0.1:9102", "127.0.0.1:9103"},
		Times:   times,
	})
	raft.Run()
}
func TestRaft2(t *testing.T) {
	raft := NewRaftWithConfig(&RConfig{
		Id:      2,
		Port:    9102,
		Cluster: []string{"127.0.0.1:9101", "127.0.0.1:9103"},
		Times:   times,
	})
	raft.Run()
}

func TestRaft3(t *testing.T) {
	raft := NewRaftWithConfig(&RConfig{
		Id:      3,
		Port:    9103,
		Cluster: []string{"127.0.0.1:9102", "127.0.0.1:9101"},
		Times:   times,
	})
	raft.Run()
}

func TestClient(t *testing.T) {
	tick := time.NewTicker(5 * time.Second)
	address := "127.0.0.1:9102"
loop:
	<-tick.C
	log.Printf("start client ----- adderss:%s\n", address)
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		log.Fatalf("create client error:%v\n", err)
	}
	args := Command{
		KeyValuePair: KeyValuePair{Key: "x", Value: "5"},
	}
	reply := &ClientResp{Success: false}
	if err := client.Call("RaftSpace.SetNewLogEntry", args, reply); err == nil {
		log.Printf("reply:%v\n", reply)
		if reply.Success {
			log.Printf("set new logEntry %v success \n", args)
			tick.Stop()
			return
		} else {
			if reply.LeaderAddress != "" {
				address = reply.LeaderAddress
				goto loop
			}
		}
		if err := client.Close(); err != nil {
			log.Printf("client close error:%v \n", err)
		}

	}
}
