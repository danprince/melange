package main

import (
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
)

//go:embed nodejs_host.js
var hostJs []byte

const sockAddr = "/tmp/melange.sock"
const tmpHostFile = "/tmp/melange.js"

var nodeConn net.Conn = nil

func getOrCreateConn() (net.Conn, error) {
	if nodeConn != nil {
		return nodeConn, nil
	}

	os.WriteFile(tmpHostFile, hostJs, os.ModePerm)

	if err := os.RemoveAll(sockAddr); err != nil {
		return nil, err
	}

	listener, err := net.Listen("unix", sockAddr)

	if err != nil {
		return nil, err
	}

	defer listener.Close()

	// TODO: Why doesn't this work?
	//cmd := exec.Command("node", "-e", fmt.Sprintf(`'%s'`, hostJs))

	cmd := exec.Command("node", tmpHostFile)
	err = cmd.Start()

	if err != nil {
		return nil, err
	}

	conn, err := listener.Accept()

	if err != nil {
		log.Fatal(err)
	}

	nodeConn = conn
	return conn, nil
}

func nodeExecFile(name string, out any) error {
	conn, err := getOrCreateConn()
	conn.Write([]byte(name))

	if err != nil {
		return err
	}

	dec := json.NewDecoder(nodeConn)
	err = dec.Decode(out)

	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
