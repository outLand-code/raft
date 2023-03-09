# Raft

  Raft is a more simple and easy to understand distributed algorithm,which mainly solves the consistency
problem in distribution.it can ensure the log is identical to other servers's log ,so it is usually used 
for distributed storage system and distributed task scheduling system.

  The best known example is probably Kubernetes, which relies on Raft through [the etcd](https://github.com/etcd-io/etcd) distributed 
key-value store.

  Read [the Raft Paper](https://raft.github.io/raft.pdf) is important for Learning Raft at first,which can help us better understand the
Raft's mechanism,and [the Raft website](https://raft.github.io/) can show more details by playing with the visualization,
and [Eli Bendersky's website](https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/) can help us how to implement it.
