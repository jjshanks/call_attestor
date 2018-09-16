package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type tcpConnWriter struct {
	*httptest.ResponseRecorder
}

type unixConnWriter struct {
	*httptest.ResponseRecorder
	bufrw *bufio.ReadWriter
	conn  net.Conn
}

func (w *tcpConnWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()
	return conn, nil, nil
}

func (w *unixConnWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	tmpdir, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return nil, nil, err
	}
	tmpFilePath := filepath.Join(tmpdir, "testrun"+strconv.FormatInt(time.Now().UnixNano(), 10))
	_, err = net.Listen("unix", tmpFilePath)
	if err != nil {
		return nil, nil, err
	}
	w.conn, err = net.Dial("unix", tmpFilePath)
	if err != nil {
		return nil, nil, err
	}
	w.bufrw = bufio.NewReadWriter(bufio.NewReader(w.Body), bufio.NewWriter(w.Body))
	return w.conn, w.bufrw, nil
}

func TestHijackFailure(t *testing.T) {
	w := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	handler := &AttestorSocketHandler{}
	handler.ServeHTTP(w, request)
	response := w.Result()
	if response.StatusCode != 500 {
		t.Errorf("Expected status code 500 but got %d", response.StatusCode)
	}
	body, _ := ioutil.ReadAll(response.Body)
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "webserver doesn't support hijacking" {
		t.Errorf("Expected response body of 'webserver doesn't support hijacking' but got '%s'", bodyStr)
	}
}

func TestUnixConnCastFailure(t *testing.T) {
	w := &tcpConnWriter{httptest.NewRecorder()}
	request := httptest.NewRequest("GET", "/", nil)
	handler := &AttestorSocketHandler{}
	handler.ServeHTTP(w, request)
	response := w.Result()
	if response.StatusCode != 500 {
		t.Errorf("Expected status code 500 but got %d", response.StatusCode)
	}
	body, _ := ioutil.ReadAll(response.Body)
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "connection from webserver isn't a UnixConn" {
		t.Errorf("Expected response body of 'connection from webserver isn't a UnixConn' but got '%s'", bodyStr)
	}

}

func TestValidResponse(t *testing.T) {
	w := &unixConnWriter{ResponseRecorder: httptest.NewRecorder()}
	request := httptest.NewRequest("GET", "/", nil)
	handler := &AttestorSocketHandler{}
	handler.ServeHTTP(w, request)
	response := w.Result()
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200 but got %d", response.StatusCode)
	}
	body, _ := ioutil.ReadAll(response.Body)
	bodyStr := strings.TrimSpace(string(body))
	pid := os.Getpid()
	user, _ := user.Current()
	expected := fmt.Sprintf("Hi there, you are pid %d, uid %s, and name %s!", pid, user.Uid, user.Username)
	if bodyStr != expected {
		t.Errorf("Expected response body of '%s' but got '%s'", expected, bodyStr)
	}
}
