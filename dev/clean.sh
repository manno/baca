#!/bin/bash
set -e
rm -f bca backend.test
go clean -testcache
