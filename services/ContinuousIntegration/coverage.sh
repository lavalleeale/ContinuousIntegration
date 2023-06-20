#!/usr/bin/env bash
rm -rf covdatafiles &&
    mkdir covdatafiles &&
    go build -cover . &&
    GOCOVERDIR=covdatafiles ./ContinuousIntegration &&
    go tool covdata percent -i=covdatafiles &&
    go tool covdata textfmt -i=covdatafiles -o covdatafiles/profile.txt &&
    go tool cover -html=covdatafiles/profile.txt -o covdatafiles/cover.html
