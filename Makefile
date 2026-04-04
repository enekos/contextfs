SHELL := /bin/bash

.DEFAULT_GOAL := help

.PHONY: help install install-dashboard setup build typecheck lint test clean dashboard dashboard-api dashboard-dev mairu-build mairu-web
.PHONY: meili-up meili-down meili-status meili-clean setup-no-docker dev-no-docker mairu-no-docker

help:
	@echo "mairu monorepo Makefile"
	@echo
	@echo "Core:"
	@echo "  make install            Install root Bun dependencies"
	@echo "  make install-dashboard  Install dashboard dependencies"
	@echo "  make build              Build TypeScript output"
	@echo "  make typecheck          Run TypeScript typecheck"
	@echo "  make lint               Run oxlint"
	@echo "  make test               Run Vitest suite"
	@echo "  make clean              Remove dist artifacts"
	@echo
	@echo "Runtime:"
	@echo "  make setup              Initialize indexes (requires Meilisearch)"
	@echo "  make dashboard          Start dashboard API + UI"
	@echo "  make mairu-build        Build mairu binary"
	@echo "  make mairu-web          Start mairu web UI (expects binary)"
	@echo
	@echo "No Docker (self-contained Meilisearch fallback):"
	@echo "  make meili-up           Download/run local Meilisearch"
	@echo "  make meili-down         Stop local Meilisearch"
	@echo "  make meili-status       Show local Meilisearch status"
	@echo "  make meili-clean        Stop and wipe local Meilisearch data"
	@echo "  make setup-no-docker    Install deps, start local Meilisearch, setup indexes"
	@echo "  make dev-no-docker      Start local Meilisearch + dashboard"
	@echo "  make mairu-no-docker    Build mairu and run web UI with local Meilisearch"

install:
	bun install

install-dashboard:
	bun install --cwd mairu/ui

setup:
	bun run setup

build:
	bun run build

typecheck:
	bun run typecheck

lint:
	bun run lint

test:
	bun run test

clean:
	bun run clean

dashboard-api:
	bun run dashboard:api

dashboard-dev:
	bun run dashboard:dev

dashboard:
	bun run dashboard

mairu-build:
	bun run mairu:build

mairu-web:
	bun run mairu:web

meili-up:
	./mairu/contextfs/scripts/meili-local.sh up

meili-down:
	./mairu/contextfs/scripts/meili-local.sh down

meili-status:
	./mairu/contextfs/scripts/meili-local.sh status

meili-clean:
	./mairu/contextfs/scripts/meili-local.sh clean

setup-no-docker: install install-dashboard meili-up setup

dev-no-docker: meili-up
	bun run dashboard

mairu-no-docker: meili-up mairu-build
	./mairu/bin/mairu-agent web -p 8080
