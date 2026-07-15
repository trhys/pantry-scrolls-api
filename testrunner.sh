#!/bin/sh
docker compose run --rm test-runner go test ./... -count=1
