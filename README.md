# Call Attestor
This is an example of using a unix socket to validate a caller.

## Build
`go build call_attestor.go`

## Running Attestor
`$ ./call_attestor -socket_path /tmp/my_socket`

## Calling Attestor
In this example the caller is called testuser with uid 1001.
```
$ curl --unix-socket /tmp/my_socket http://whatever &
[1] 20740
$ Hi there, you are pid 20740, uid 1001, and name testuser!
```
