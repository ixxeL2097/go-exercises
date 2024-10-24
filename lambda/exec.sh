#!/bin/bash

export GOPROXY=direct
export GONOPROXY="proxy.golang.org/*"
export GOPRIVATE="golang.org,google.golang.org,gopkg.in,k8s.io,sum.golang.org,sigs.k8s.io,mvdan.cc,github.com,honnef.co,nullprogram.com,rsc.io"
export GOINSECURE="golang.org,google.golang.org,gopkg.in,k8s.io,sum.golang.org,sigs.k8s.io,mvdan.cc,github.com,honnef.co,nullprogram.com,rsc.io"
go -C . mod tidy
go -C . run lambda.go
# go install -v golang.org/x/tools/gopls@latest
# go install -v golang.org/x/tools/cmd/goimports@latest