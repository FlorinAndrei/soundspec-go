package main

// Version is set at build time via ldflags:
// go build -ldflags "-X main.Version=$(date -u +%Y%m%d-%H%M%S)"
var Version = "dev"
