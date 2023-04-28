package raft

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"sync"
	"time"
)

type Storage struct {
	mu   sync.Mutex
	data map[string][]byte
}
type storeData struct {
	Term    int
	VoteFor int
	Log     []LogEntry
}

func NewStorage() (store *Storage) {
	store = &Storage{data: make(map[string][]byte)}
	return
}

func (r *Raft) persistent() {
	sd := &storeData{
		Term:    r.currentTerm,
		VoteFor: r.votedFor,
		Log:     r.log,
	}
	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(sd); err != nil {
		log.Printf("the Raft store data error :%v\n", err)
	}
	r.storage.data["currentTerm"] = data.Bytes()
	log.Printf("the storage data :%v\n", r.storage.data)
}

func (r *Raft) loadFromStorage() {
	r.storage.mu.Lock()
	defer r.storage.mu.Unlock()
	if byt, err := os.ReadFile(Config.StorePath + "/storage.rfdb"); err == nil && len(byt) > 0 {
		var sd storeData
		if err := gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&sd); err != nil {
			log.Printf("the Raft load store data error:%v\n", err)
		} else {
			log.Printf("the Raft load data from storage term:%d ,voteForId:%d,length of log:%d\n",
				sd.Term, sd.VoteFor, len(sd.Log))
			r.currentTerm = sd.Term
			r.votedFor = sd.VoteFor
			r.log = sd.Log
		}
	}
}

func (s *Storage) persistentToDisk() {
	s.mu.Lock()
	defer s.mu.Unlock()
	tick := time.NewTicker(time.Duration(Config.StoreTime) * time.Second)
	for {
		<-tick.C
		if data, exist := s.data["currentTerm"]; exist {
			log.Println("start persistent to disk ")
			store := func() error {
				file, err := os.OpenFile(Config.StorePath+"/storage.rfdb", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
				defer file.Close()
				if err != nil {
					return err
				}
				_, _ = file.Write(data)
				return nil
			}
			if err := store(); err != nil {
				log.Printf("store file error:%v\n", err)
				continue
			}

		}
	}
}
