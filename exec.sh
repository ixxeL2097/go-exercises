#!/bin/bash

export GOPROXY=direct
export GONOPROXY="proxy.golang.org/*"
export GOPRIVATE="golang.org,google.golang.org,gopkg.in,k8s.io,sum.golang.org,sigs.k8s.io,mvdan.cc,github.com,honnef.co"
export GOINSECURE="golang.org,google.golang.org,gopkg.in,k8s.io,sum.golang.org,sigs.k8s.io,mvdan.cc,github.com,honnef.co" 
go install -v golang.org/x/tools/gopls@latest
go install -v golang.org/x/tools/cmd/goimports@latest