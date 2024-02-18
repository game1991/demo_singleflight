package main

//go:generate struct2interface -d pkg/redis
//go:generate struct2interface -d pkg/http
//go:generate wire ./...
