#!/bin/bash


go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --go_out=.  --go-grpc_out=. holloman.proto
mv github.com/wessorh/HuntingHash/holloman* .

git clone https://github.com/bamiaux/rez
cd rez
go mod init github.com/bamiaux/rez
go mod tidy
cd ../cmd
cp ../LICENSE.md .
go build .
cd ../curve
go build .
./curve -compress -file ../cmd/hilbert_curve.dat.gz -mode generate -order 15 -verbose
cd ../cmd
./cmd -rest-port :50005 -debug -v
