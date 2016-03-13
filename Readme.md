# locker

This is a basic client/server system to lock files on the filesystem level.

Note: The code here is _as is_, it's not nice and violating many golang programming patterns, so be careful.

# Background

I wanted a system to consistently backup Mac .sparsebundle files on a linux box.

In my case, this couldn't be done by filesystem snapshots, as LVM2 can't snapshot SSD cached volumes.

As i knew, Mac OS X can prevent concurrent access, i took apart the OS X/netatalk behaviour and found, that a simple file lock on the `token` is sufficient to prevent access to the sparsebundle. If your sparsebundle is encrypted, make sure to create a copy of the `token` as it contains the encryption key of the volume. The file is empty for unencrypted volumes.

# Building

Check it out on a system with a go installation.

## the dirty way

build: `go build`

run: `go run *.go $parameters`

## the nice way

The directory structure should be: `$dir/src/beck/locker` and `$dir` will be used as `GOPATH`.

There's a `Makefile` included, which will create a build for `linux-amd64`, so just do a `make build` and the `locker` binary will be created.

If you're not on linux, just remove the `GOOS` and `GOARCH` parameters from the `Makefile`

# Installation

- Drop the built binary in a path like `/usr/local/bin`
- Copy `locker.sysconfig` to `/etc/sysconfig/locker`
- Install `locker.service` to `/usr/lib/systemd/system/locker.service`
- Enable the service using `systemctl enable locker`
- Start the server: `systemctl start locker`

Whitelisted entries should be specified using the `OPTIONS` in the sysconfig file like
`OPTIONS="--allow \"/mnt/storage/TimeMachine/lobeck’s MacBook Pro.sparsebundle/token\"`

# Usage

This consists of two parts, a server which is holding the lock and a client which requests locks and unlocks of files.

The communication between the processes is done over unix sockets, the socket is created with mode 0600 to limit access for other users.

Additionally, there's a whitelist of lockable files. Both the `-allow` and `-path` parameters support relative paths.

For proper automation, the client uses exit code 0 if successful, otherwise 1

    $ ./locker -h
      --allow value
          allow path (multiple times usable) (default [])
      --client
          act as client
      -h  show help
      --help
          show help
      --lock
          lock file
      --path string
          path to lock
      --server
          act as server
      --socket string
          socket path (default "/var/run/locker.sock")
      --unlock
          unlock file

Start the server:

    $ ./locker --server --socket /tmp/echo.sock --allow foo --allow bar.sparsebundle/token
    2016/03/13 03:59:05 Current whitelist: [/Users/beck/foo /Users/beck/bar.sparsebundle/token]
    2016/03/13 03:59:05 server ready

Request a lock:

The call is blocking until the lock is acquired, so this might take a while, if the file/sparsebundle is currently in use

    $ ./locker -client --socket /tmp/echo.sock -lock -path bar.sparsebundle/token
    ok - lock set
    $ ./locker -client --socket /tmp/echo.sock -lock -path bar.sparsebundle/token
    ok - file already locked

If you now try to mount the sparsebundle, you'll get a "Resource temporarily unavailable message", this is also properly handled by TimeMachine.

Release a lock:

    $ ./locker -client --socket /tmp/echo.sock -unlock -path bar.sparsebundle/token
    ok - lock released
    $ ./locker -client --socket /tmp/echo.sock -unlock -path bar.sparsebundle/token
    ok - file is not locked
