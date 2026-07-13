#!/bin/sh
./goose -dir sql/schema postgres $DB up
./seed
