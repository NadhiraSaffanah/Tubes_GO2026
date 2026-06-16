package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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

	serverReader := bufio.NewReader(conn)
	prompt, err := serverReader.ReadString('\n')
	if err != nil {
		fmt.Println("Connection closed")
		return
	}

	fmt.Print(prompt)

	keyboard := bufio.NewReader(os.Stdin)

	username, err := keyboard.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read username")
		return
	}

	fmt.Fprint(conn, username)

	// baca respon server
	response, err := serverReader.ReadString('\n')
	if err != nil {
		fmt.Println("Connection closed")
		return
	}

	response = strings.TrimSpace(response)

	if response == "USERNAME_ALREADY_EXISTS" {
		fmt.Println("Username already exists!")
		return
	}

	fmt.Println("Login successful!")

	// goroutine untuk menerim pesan dari server
	go func() {
		for {
			serverMsg, err := serverReader.ReadString('\n')

			// handle error, jika server menutup koneksi, client akan keluar
			if err != nil {
				fmt.Println("\nServer closed the connection.")
				os.Exit(0) // paksa seluruh client berhenti saat ini juga
			}

			serverMsg = strings.TrimSpace(serverMsg)

			if serverMsg == "TIMEOUT" {
				fmt.Println("\nDisconnected from server")
				os.Exit(0)
			}

			fmt.Printf("\r\033[K%s\n> ", serverMsg) // \033[K = hapus sisa baris
		}
	}()

	// kirim pesan
	for {
		//fmt.Print("> ")

		msg, err := keyboard.ReadString('\n')

		if err != nil {
			return
		}

		fmt.Fprint(conn, msg)
		fmt.Print("> ")
	}
}
