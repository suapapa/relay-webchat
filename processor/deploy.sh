#!/bin/bash

GOOS=linux GOARCH=arm64 go build -o processor_homin-dev
scp processor_homin-dev vraptor@192.168.219.140:~/
# ssh suapapa@192.168.219.146 "cd ~/ && ./processor_homin-dev"
