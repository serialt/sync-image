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
 
# ========================
# Project Info
# ========================
PROJECT_NAME := sync-image
APP_NAME     := $(PROJECT_NAME)
DIST_DIR     := dist

GOBASE      := $(shell pwd)
GOFILES     := $(wildcard *.go)

# ========================
# Version & Build Info
# ========================
BRANCH      := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION  := $(shell go version | awk '{print $$3}')
KEY         := wzFdVccccccccccccccc

LDFLAGS := -s -w \
	-X 'main.APPVersion=$(BRANCH)' \
	-X 'main.GoVersion=$(GO_VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GitCommit=$(GIT_HASH)' \
	-X 'main.AesKey=$(KEY)'

PLATFORMS := \
	linux/amd64 \
	linux/arm64 
# 	linux/riscv64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.PHONY: all
all: build

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)

.PHONY: serve
serve:
	@echo "Running locally..."
	go run .

.PHONY: build
build: clean
	@mkdir -p $(DIST_DIR)
	@echo "Building $(APP_NAME) for local OS..."
	@go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)
	@echo "\n✅ Build succeeded:"
	@ls -lh $(DIST_DIR)/$(APP_NAME)*

.PHONY: release
release: clean
	@mkdir -p $(DIST_DIR)
	@go mod tidy
	@set -e; \
	for plat in $(PLATFORMS); do \
		OS=$${plat%/*}; \
		ARCH=$${plat#*/}; \
		EXE=$$( [ "$$OS" = "windows" ] && echo ".exe" || echo "" ); \
		OUT=$(DIST_DIR)/$(APP_NAME)-$$OS-$$ARCH$$EXE; \
		echo "🚀 Building $$OUT"; \
		GOOS=$$OS GOARCH=$$ARCH go build -trimpath -ldflags "$(LDFLAGS)" -o $$OUT .; \
	done
	@echo "\n✅ Release build succeeded:"
	@ls -lh $(DIST_DIR)/
