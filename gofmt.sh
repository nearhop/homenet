#!/bin/sh

 if [ "$(find . -iname '*.go' | grep -v '\.pb\.go$' | xargs goimports -l)" ]
  then
    find . -iname '*.go' | grep -v '\.pb\.go$' | xargs goimports -d
    exit 1
  fi
