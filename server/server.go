package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// untuk menyimpan username client
	clients = make(map[string]net.Conn)
	// map menyimpan di room apa ada user siapa aja. Struct hanya untuk membuat net.Conn menjadi array tapi bentuk map
	roomClient = make(map[int]map[net.Conn]struct{})
	// map menyimpan user sedang di room apa
	clientRoom = make(map[net.Conn]int)
	// slice menyimpan id room dengan nama room, dengan index di array bersifat sebagai id
	roomArr []string
	mutex   sync.Mutex
)

const leaveCmd = "/leave"
const createCmd = "/create"
const endCmd = "/quit"

func main() {
	// membuka koneksi
	ln, err := net.Listen("tcp", ":9090")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen!\n")
		os.Exit(1)
	} else {
		fmt.Println("Listening on port 9090...")
	}

	// karena banyak client, penerimaan koneksi di-looping
	for {
		// terima koneksi
		conn, err := ln.Accept()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept!\n") // kalau gagal
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
	clientRoom[conn] = -1

	mutex.Unlock()

	fmt.Printf("%s connected\n", username)
	fmt.Fprintln(conn, "WELCOME")

	welcomeBroadcastMessage := fmt.Sprintf("[SERVER] %s has entered the server!!", username)
	broadcastMessage(welcomeBroadcastMessage, conn)

	defer func() {
		mutex.Lock()

		//delete user dari semua map
		roomId := clientRoom[conn]
		if roomId != -1 { //jika user bukan di lobby/ bukan di room apapun
			delete(roomClient[roomId], conn)
		}
		delete(clients, username)
		delete(clientRoom, conn)

		mutex.Unlock()
		if roomId != -1 { // ini dipisah karena kalau di diatas, bakal kena mutex.Lock() 2 kali
			tmp := fmt.Sprintf("user [%s] has left the room", username)
			messageRoom(tmp, roomId)
		}
		fmt.Printf("%s disconnected\n", username)
		disconectBroadcastMessage := fmt.Sprintf("[SERVER] %s has disconnected from the server.", username)
		broadcastMessage(disconectBroadcastMessage, nil)
	}()

	fmt.Fprintf(conn, "To close your connection use command %s\n", endCmd)
	fmt.Fprintf(conn, "To leave the room use command %s\n", leaveCmd)
	fmt.Fprintf(conn, "To create a room use command %s\n", createCmd)
	for {
		// pasang timeout 3 menit
		// jika tidak ada jawaban dalam 3 menit, tutup koneksi
		conn.SetReadDeadline(time.Now().Add(3 * time.Minute))

		//jika tidak ada room minta user pilih salah satu room dahulu
		mutex.Lock()
		roomId := clientRoom[conn]
		mutex.Unlock()
		if roomId == -1 {
			displayRooms(conn)
			message, err := reader.ReadString('\n') // baca pesan dengan bufio sampai ketemu newline

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Fprintln(conn, "TIMEOUT")
				}

				return
			}

			message = strings.TrimSpace(message)
			//cek apakah user mau stop koneksi
			if endCmd == message {
				return
			}

			//cek apakah user mau buat room baru
			if createCmd == message {
				fmt.Fprintln(conn, "Enter a unique name for the room(name will be converted to lower case):")
				roomName, err := reader.ReadString('\n') // baca pesan dengan bufio sampai ketemu newline

				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						fmt.Fprintln(conn, "TIMEOUT")
					}

					return
				}

				roomName = strings.TrimSpace(roomName)
				roomName = strings.ToLower(roomName)
				if createRoom(roomName) == true {
					fmt.Fprintln(conn, "[SERVER] Room has been created")
					continue
				} else if roomName == "" {
					fmt.Fprintln(conn, "[SERVER] Room name cannot be empty")
					continue
				} else {
					fmt.Fprintln(conn, "[SERVER] Name for room has already been taken")
					continue
				}
			}

			//konversi message ke integer
			num, err := strconv.Atoi(message)

			//cek apakah waktu di convert ke integer gagal atau berhasil
			if err != nil {
				fmt.Fprintln(conn, "Invalid number, please type in an integer!(1, 2, 3, ....)")
				continue
			}

			//cek apakah input out of bounds
			mutex.Lock()
			roomArrLength := len(roomArr)
			mutex.Unlock()
			if num <= roomArrLength && num > 0 {
				joinRoom(conn, num-1)
				tmp := fmt.Sprintf("user [%s] has joined the room", username)
				mutex.Lock()
				roomId = clientRoom[conn]
				mutex.Unlock()
				messageRoom(tmp, roomId)
			} else {
				fmt.Fprintln(conn, "A room with that id doesn't exists, please try again.")
				continue
			}

			fmt.Fprintln(conn, "")
			fmt.Fprintln(conn, "")
			fmt.Fprintln(conn, "")
			fmt.Fprintln(conn, "")

		} else { //Setelah room sudah dipilih baru masuk ke else
			message, err := reader.ReadString('\n') // baca pesan dengan bufio sampai ketemu newline

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Fprintln(conn, "TIMEOUT")
				}

				return
			}

			message = strings.TrimSpace(message)
			//cek apakah user type command untuk leave room
			if leaveCmd == message {
				leaveRoom(conn, roomId)
				tmp := fmt.Sprintf("user [%s] has left the room", username)
				messageRoom(tmp, roomId)
				continue
			}
			//cek apakah user mau stop koneksi
			if endCmd == message {
				return
			}

			message = fmt.Sprintf("[%s]: %s", username, message)
			messageRoom(message, roomId)
		}
	}
}

func broadcastMessage(msg string, exclude net.Conn) {
	//Fungsi untuk membroadcast ke semua user(tidak penting room), fungsi tidak wajib no 2
	mutex.Lock()
	defer mutex.Unlock()

	for _, conn := range clients {
		if exclude != conn {
			fmt.Fprintln(conn, msg)
		}
	}
}

func messageRoom(msg string, room int) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn := range roomClient[room] {
		fmt.Fprintln(conn, msg)
	}
}

func joinRoom(conn net.Conn, roomId int) {
	mutex.Lock()
	defer mutex.Unlock()
	clientRoom[conn] = roomId
	roomClient[roomId][conn] = struct{}{}
}

func leaveRoom(conn net.Conn, roomId int) {
	mutex.Lock()
	defer mutex.Unlock()
	clientRoom[conn] = -1
	delete(roomClient[roomId], conn)
}

func displayRooms(conn net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	if len(roomArr) == 0 {
		fmt.Fprintf(conn, "No room exists yet, please create one with command %s\n", createCmd)
		return
	}
	for i := 0; i < len(roomArr); i++ {
		fmt.Fprintf(conn, "%d. [%s], %d users\n", i+1, roomArr[i], len(roomClient[i]))
	}
	fmt.Fprintln(conn, "Pick one of the rooms by number, to start chatting!")
}

func createRoom(name string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	for _, room := range roomArr {
		if room == name {
			return false
		}
	}
	roomArr = append(roomArr, name)
	newRoomId := len(roomArr) - 1
	roomClient[newRoomId] = make(map[net.Conn]struct{})
	return true
}
