package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// Buat koneksi ke server menggunakan net.Dial
	conn, err := net.Dial("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!")
	} else {
		fmt.Println("Connected to server!")
	}
	// Kalau client selesai menggunakan program, koneksi akan ditutup
	defer conn.Close()

}
