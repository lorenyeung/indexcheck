#!/bin/bash
rm indexcheck
go build -o indexcheck
cp indexcheck ~/.jfrog/plugins/
