@echo off

set BUILD_OPTIONS=-v

::format source files
go fmt ./...

::build
if not exist build (
    mkdir build
)
pushd build
go build -o ./workers.exe  %BUILD_OPTIONS% ../worker.go ../cli.go ../generator.go ../main.go ../common.go
popd 

echo Build successfull.

