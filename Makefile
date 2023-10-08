# ******************************************************
# Author       	:	serialt 
# Email        	:	tserialt@gmail.com
# Filename     	:   Makefile
# Version      	:	v1.3.0
# Created Time 	:	2021-06-25 10:47
# Last modified	:	2023-07-05 19:20
# By Modified  	: 
# Description  	:       build go package
#  
# ******************************************************


PROJECT_NAME= sync-image


GOBASE=$(shell pwd)
GOFILES=$(wildcard *.go)


BRANCH := $(shell git symbolic-ref HEAD 2>/dev/null | cut -d"/" -f 3)
# BRANCH := `git fetch --tags && git tag | sort -V | tail -1`
# BUILD := $(shell git rev-parse --short HEAD)
BUILD_DIR := $(GOBASE)/dist
VERSION = $(BRANCH)

BuildTime := $(shell date -u  '+%Y-%m-%d %H:%M:%S %Z')
GitHash := $(shell git rev-parse HEAD)
GoVersion = $(shell go version | cut -d " " -f 3 )
Maintainer := ""
KEY := ""

PKGFLAGS := " -s -w -X 'main.APPVersion=$(VERSION)' -X 'main.GoVersion=$(GoVersion)'  -X 'main.BuildTime=$(BuildTime)' -X 'main.GitCommit=$(GitHash)' "

APP_NAME = $(PROJECT_NAME)
# go-pkg.v0.1.1-linux-amd64

.PHONY: clean
clean:
	@-rm -rf dist/$(PROJECT_NAME)* 

.PHONY: serve
serve:
	go run .

.PHONY: build
build: clean
	@go build -trimpath -ldflags $(PKGFLAGS) -o "dist/$(APP_NAME)" 
	@echo "\n******************************"
	@echo "         build succeed "
	@echo "******************************\n"
	@ls -la dist/$(PROJECT_NAME)*
	@echo

.PHONY: build-linux
build-linux: clean
	@go mod tidy
	@GOOS="linux"   GOARCH="amd64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-linux-amd64"       
	@GOOS="linux"   GOARCH="arm64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-linux-arm64"    
	@echo "\n******************************"
	@echo "      build linux succeed "
	@echo "******************************\n"
	@ls -la dist/$(PROJECT_NAME)*
	@echo

.PHONY: release
release: clean
	@go mod tidy
	@GOOS="windows" GOARCH="amd64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-windows-amd64.exe" 
	@GOOS="linux"   GOARCH="amd64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-linux-amd64"       
	@GOOS="linux"   GOARCH="arm64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-linux-arm64"       
	@GOOS="darwin"  GOARCH="amd64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-darwin-amd64"      
	@GOOS="darwin"  GOARCH="arm64" go build -trimpath -ldflags $(PKGFLAGS) -v -o "dist/$(APP_NAME)-darwin-arm64"      
	@echo "\n******************************"
	@echo "        release succeed "
	@echo "******************************\n"
	@ls -la dist/$(PROJECT_NAME)*
	@echo