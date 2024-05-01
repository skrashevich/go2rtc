#!/bin/sh
ORIGDIR=$(pwd)
for i in stats; do 
cd plugins/$i
go build -buildmode=plugin -trimpath -ldflags "-s -w" -o ../$i.so plugin.go
done;
cd "$ORIGDIR"
