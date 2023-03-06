package raft

import (
	"testing"
)

var times = 100

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
