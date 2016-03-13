package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

var requestCounter int64
var allowedPaths = make(map[string]*os.File)

type StringSliceFlag []string

func (s StringSliceFlag) String() string {
	str := "["
	for i, elem := range s {
		if i > 0 {
			str += " "
		}
		str += fmt.Sprint(elem)
	}
	return str + "]"
}

func (s *StringSliceFlag) Set(value string) error {
	path, err := AbsPath(value)
	if err != nil {
		return err
	}
	*s = append(*s, path)
	return nil
}

func startServer(socketFile string, pathWhitelist StringSliceFlag) {
	for _, path := range pathWhitelist {
		allowedPaths[path] = nil
	}

	if socketExists(socketFile) {
		log.Println("existing socket detected - checking for a running server instance")
		if socketAlive(socketFile) {
			panic("socket seems live - is there another instance running?")
		} else {
			log.Println("socket seems stale - deleting it")
			syscall.Unlink(socketFile)
		}
	}

	// set permissions to to 600
	oldUmask := syscall.Umask(syscall.S_IXUSR | syscall.S_IRGRP | syscall.S_IWGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IWOTH | syscall.S_IXOTH)
	l, err := net.Listen("unix", socketFile)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()
	syscall.Umask(oldUmask) // reset umask to previous value

	log.Println("server ready")
	for {
		fd, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		d := json.NewDecoder(fd)

		atomic.AddInt64(&requestCounter, 1)
		var msg LockMessage

		err = d.Decode(&msg)
		if err != nil {
			panic(err)
		}
		responseMessage := processMessage(msg)

		log.Printf("request %d - response: %+v", requestCounter, responseMessage)
		encoder := json.NewEncoder(fd)
		encoder.Encode(responseMessage)
		fd.Close()
	}
}

func processMessage(message LockMessage) LockResponse {
	log.Printf("request %d - message: %+v\n", requestCounter, message)
	if message.Action != "lock" && message.Action != "unlock" {
		return LockResponse{Message: fmt.Sprintf("illegal action: %s", message.Action)}
	}

	filename := filepath.Clean(message.Filename)
	if !filepath.IsAbs(filename) {
		return LockResponse{Message: "need absolute path"} // indicates a broken client
	}
	if validFilename(filename) {
		switch message.Action {
		case "lock":
			fileHandle, _ := allowedPaths[filename]
			if fileHandle != nil {
				return LockResponse{Success: true, Message: "ok - file already locked"}
			}
			fileHandle, err := lockFile(filename)
			if err != nil {
				return LockResponse{Success: false, Message: err.Error()}
			}
			allowedPaths[filename] = fileHandle
			return LockResponse{Success: true, Message: "ok - lock set"}
		case "unlock":
			fileHandle, _ := allowedPaths[filename]
			if fileHandle == nil {
				return LockResponse{Success: true, Message: "ok - file is not locked"}
			}
			fileHandle.Close()
			allowedPaths[filename] = nil
			return LockResponse{Success: true, Message: "ok - lock released"}
		}

	}
	return LockResponse{Success: false, Message: fmt.Sprintf("filename %s not whitelisted", filename)}
}

func validFilename(filename string) bool {
	if _, ok := allowedPaths[filename]; ok {
		return true
	}
	return false
}

func lockFile(path string) (*os.File, error) {
	if _, err := os.Stat(path); err == nil {
		fileHandle, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		log.Printf("request %d - trying to get lock - might take a while", requestCounter)
		// see: https://gist.github.com/lobeck/033040abd74e44cce9c4
		err = syscall.FcntlFlock(fileHandle.Fd(), syscall.F_SETLKW, &syscall.Flock_t{Start: 9223372036854775799, Len: 1, Type: syscall.F_WRLCK, Whence: int16(os.SEEK_SET)})
		if err != nil {
			return nil, err
		}
		return fileHandle, nil
	} else {
		return nil, err
	}
}

func socketExists(socketFilename string) bool {
	if s, _ := os.Lstat(socketFilename); s != nil {
		return true
	}
	return false
}
func socketAlive(socketFilename string) bool {
	_, err := net.Dial("unix", socketFilename)
	switch t := err.(type) {
	case nil:
		return true
	case *net.OpError:
		return false
	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			return false
		} else {
			panic(err)
		}
	default:
		panic(err)
	}
	return false
}
