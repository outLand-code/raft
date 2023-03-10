#!/bin/sh
cd ../
for i in {1..10}
do
    echo $i
    go test -v -race -run=Raft$i
done