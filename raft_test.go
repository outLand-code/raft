package raft

import (
	"os"
	"testing"
)

func TestRaft(t *testing.T) {
	defer func() {
		_ = os.Remove(EnvKeyRaftUseLocalConfig)
	}()
	_ = os.Setenv(EnvKeyRaftUseLocalConfig, "y")
	raft := NewRaftWithConfig(&RConfig{})
	raft.Run()
}
