# Homematic-Homekit Bridge

This is a bridge to enable controlling Homematic using Homekit (iOS/Watch OS/tvOS). It is written in Go and can be cross-compiled in order to run directly on the Homematic CCU, without any additional dependencies. It only needs around 7 MB of RAM, so it is much lighter than other solutions.

## Installation

You will need Go ≥ 1.9 before starting the procedure.

	go get -u github.com/thomersch/homematic-homekit
	cd $GOPATH/github.com/thomersch/homematic-homekit # if $GOPATH is not set, it defaults to ~/go
	GOOS=linux GOARCH=arm GOARM=5 go build # this writes a binary into the same directory

After this, copy the binary onto the CCU and run it. For automatic start during boot, use the init script (homematic-homekit.sh).
