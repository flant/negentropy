#!/bin/bash

echo Generate Go sources...

(
cd vault
go generate
)

(
cd flant-server-accessd/system
minimock -i github.com/flant/server-access/flant-server-accessd/system.Interface -o ./mock/system_operator_mock_test.go
)
