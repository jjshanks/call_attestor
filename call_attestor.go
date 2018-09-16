package main

import (
    "fmt"
    "flag"
    "net"
    "net/http"
    "syscall"
    "os"
    "os/user"
    "github.com/shirou/gopsutil/process"
)

type AttestorSocketHandler struct{}

func (h *AttestorSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	highjacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := highjacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Don't forget to close the connection:
	defer conn.Close()
	uconn, ok := conn.(*net.UnixConn)
	if !ok {
		http.Error(w, "connection from webserver isn't a UnixConn", http.StatusInternalServerError)
		return
	}

	file, err := uconn.File()
	if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
	}
	defer file.Close()

	ucred, err := syscall.GetsockoptUcred(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
	}

	pid := int32(ucred.Pid)

	proc, _ := process.NewProcess(pid)
	uids, err := proc.Uids()
	if err != nil {
		fmt.Fprintf(bufrw, err.Error())
	}
	u, err := user.LookupId(fmt.Sprint(uids[1]))
	fmt.Fprintf(bufrw, "Hi there, you are pid %d, uid %d, and name %s!", pid, uids[1], u.Username)
	fmt.Fprintln(bufrw)
	bufrw.Flush()
}

func main() {
	server := http.Server {
		Handler: &AttestorSocketHandler{},
	}
	socketPath := flag.String("socket_path", "/tmp/foo/bar", "path for socket")
	flag.Parse()
	os.Remove(*socketPath)
        unixListener, err := net.Listen("unix", *socketPath)
	defer unixListener.Close()
	os.Chmod(*socketPath, 0777)
        if err != nil {
                panic(err)
        }
        server.Serve(unixListener)
}
