package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// untuk menyimpan username client
var (
	clients = make(map[string]net.Conn)
	mutex   sync.Mutex
)

func main() {
	// membuka koneksi
	ln, err := net.Listen("tcp", ":9090")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen!")
		os.Exit(1)
	} else {
		fmt.Println("Listening on port 9090...")
	}

	// karena banyak client, penerimaan koneksi di-looping
	for {
		// terima koneksi
		conn, err := ln.Accept()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept!") // kalau gagal
			os.Exit(1)
		} else {
			fmt.Println("New connection accepted!") // kalau berhasil
		}

		go handleConnection(conn)
	}
}

// fungsi untuk menangani username duplikat
func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn) // inisialisasi reader

	// baca username
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read username")
		return
	}

	username = strings.TrimSpace(username)

	mutex.Lock() // lock dulu agar tidak berubah

	_, exists := clients[username]
	if exists { // kalau username sudah ada
		mutex.Unlock() // unlock

		fmt.Fprintln(conn, "Username already exists!")
		fmt.Printf("Rejected username: %s\n", username)

		return
	}

	// kalau username belum ada di daftar, bisa ditambahkan
	clients[username] = conn

	mutex.Unlock()

	fmt.Printf("%s connected\n", username)
	fmt.Fprintln(conn, "WELCOME")

	defer func() {
		mutex.Lock()
		delete(clients, username)
		mutex.Unlock()

		fmt.Printf("%s disconnected\n", username)
	}()

	for {
		message, err := reader.ReadString('\n') // baca pesan dengan bufio sampai ketemu newline

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read message!")
		}

		fmt.Printf("%s: %s\n", username, message)
	}
}
