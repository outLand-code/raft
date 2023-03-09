# Raft
        Raft is a more simple and easy to understand distributed algorithm,which mainly solves the consistency
    problem in distribution.it can ensure the log is identical to other servers's log ,so it is usually used 
    for distributed storage system and distributed task scheduling system.

        The best known example is probably Kubernetes, which relies on Raft through the etcd distributed 
    key-value store.

        Read the Raft Paper is important for Learning Raft at first,which can help us better understant the
    Raft's mechanism,and the Raft website can show more details by playing with the visualization,
    and Eli Bendersky's website can help us how to implement it.
