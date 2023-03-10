package main

import "raft"

func main() {
	//load configuration from app.yml and create a Raft server
	raft.NewRaft().Run()
}
