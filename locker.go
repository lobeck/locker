package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

type LockMessage struct {
	Action   string `json:"action"`
	Filename string `json:"filename"`
}

type LockResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	var help, client, server, lockAction, unlockAction bool
	var socketFilename, path string
	var allowedPaths = make(StringSliceFlag, 0)

	flag.BoolVar(&help, "help", false, "show help")
	flag.BoolVar(&help, "h", false, "show help")

	flag.BoolVar(&client, "client", false, "act as client")
	flag.BoolVar(&server, "server", false, "act as server")

	flag.BoolVar(&lockAction, "lock", false, "lock file")
	flag.BoolVar(&unlockAction, "unlock", false, "unlock file")

	flag.StringVar(&socketFilename, "socket", "/var/run/locker.sock", "socket path")

	flag.Var(&allowedPaths, "allow", "allow path (multiple times usable)")
	flag.StringVar(&path, "path", "", "path to lock")

	flag.Parse()
	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if client && server {
		log.Fatal("parameter conflict, -client and -server can't be specified together")
	}
	// default: client mode
	if !client && !server {
		client = true
	}

	if lockAction && unlockAction {
		log.Fatal("parameter conflict, -lock and -unlock can't be specified together")
	}
	if client && !lockAction && !unlockAction {
		log.Fatal("-lock or -unlock are required")
	}

	if server {
		if len(allowedPaths) == 0 {
			log.Fatal("no paths whitelisted")
		}
		log.Printf("Current whitelist: %v", allowedPaths.String())

		startServer(socketFilename, allowedPaths)
	}
	if client {
		var action string
		var msg LockResponse

		if lockAction {
			action = "lock"
		}
		if unlockAction {
			action = "unlock"
		}
		if path == "" {
			log.Fatal("path is required")
		}
		path = filepath.Clean(path)
		absPath, err := AbsPath(path)
		if err != nil {
			log.Fatal(err)
		}
		message := LockMessage{Action: action, Filename: absPath}
		c, err := net.Dial("unix", socketFilename)
		if err != nil {
			panic(err)
		}
		defer c.Close()


		encoder := json.NewEncoder(c)
		encoder.Encode(message)

		decoder := json.NewDecoder(c)
		err = decoder.Decode(&msg)
		if err != nil {
			panic(err)
		}

		fmt.Println(msg.Message)
		if !msg.Success {
			os.Exit(1)
		}
	}
}

func AbsPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	} else {
		return path, nil
	}
}
