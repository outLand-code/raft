#!/bin/sh
cd ../
# shellcheck disable=SC2039
for i in {1..3}
do
    echo "$i"
    sleep 2
    go test -v -race -run=Raft$$i
done