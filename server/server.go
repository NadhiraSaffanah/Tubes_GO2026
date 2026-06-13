package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
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

		// goroutine
		go handleConnection(conn)
	}
}

// fungsi untuk menangani username duplikat
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// pasang timeout 3 menit
	// jika tidak ada jawaban dalam 3 menit, tutup koneksi
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))

	reader := bufio.NewReader(conn)
	fmt.Fprintln(conn, "Username:") // meminta username

	// baca username
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Connection timeout")
		return
	}

	username = strings.TrimSpace(username) // ambil username

	conn.SetReadDeadline(time.Time{}) // waktu timeout di-reset

	mutex.Lock() // lock dulu agar tidak berubah
	_, exists := clients[username]

	if exists { // kalau username sudah ada
		mutex.Unlock() // unlock

		fmt.Fprintln(conn, "USERNAME_ALREADY_EXISTS")
		fmt.Printf("Rejected username: %s\n", username)

		return
	}

	// kalau username belum ada di daftar, bisa ditambahkan
	clients[username] = conn

	mutex.Unlock()

	// melakukan broadcast ketika ada user baru yang join ke percakapan
	mutex.Lock()
	for clientName, clientConnection := range clients {
		if clientName != username {
			fmt.Fprintf(clientConnection, "%s has joined the chat\n", username)
		}
	}
	mutex.Unlock()

	fmt.Printf("%s connected\n", username)
	fmt.Fprintln(conn, "WELCOME")

	defer func() {
		mutex.Lock()
		delete(clients, username)

		//melakukan broadcast ketika ada user yang disconnect dari percakapan
		for _, clientConnection := range clients {
        	fmt.Fprintf(clientConnection, "%s has left the chat\n", username)
    	}
		mutex.Unlock()

		fmt.Printf("%s disconnected\n", username)
	}()

	for {
		// pasang timeout 3 menit
		// jika tidak ada jawaban dalam 3 menit, tutup koneksi
		conn.SetReadDeadline(time.Now().Add(3 * time.Minute))

		message, err := reader.ReadString('\n') // baca pesan dengan bufio sampai ketemu newline

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Fprintln(conn, "TIMEOUT")
			}

			return
		}

		message = strings.TrimSpace(message)
		fmt.Printf("[%s]: %s\n", username, message)

		mutex.Lock()
		for clientName, clientConnection := range clients{
			if clientName != username {
				fmt.Fprintf(clientConnection, "[%s]: %s\n", username, message)
			}
		}
		mutex.Unlock()
	}
}
