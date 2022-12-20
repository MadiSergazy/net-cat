package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const filename = "welcome.txt"

var (
	historyOfMessages string
	countOfClients    = 0
	clients           = make(map[string]net.Conn)
	leaving           = make(chan message)
	messages          = make(chan message)
	mutex             sync.Mutex
)

type message struct {
	time     string
	userName string
	text     string
	address  string
}

func main() {
	args := os.Args[1:]
	port := ""
	lenArgs := len(args)
	switch lenArgs {
	case 0:
		port = "9090"
	case 1:
		_, err := strconv.Atoi(args[0])
		if ErrorHandler(err) != nil {
			return
		}
		port = args[0]
	default:
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	fmt.Println("listening on the PORT: ", port)
	listen, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	defer listen.Close()

	go broadcaster()

	for {

		conn, err := listen.Accept()

		mutex.Lock()
		countOfClients++
		fmt.Println("Clients join: ", countOfClients)
		c := countOfClients
		mutex.Unlock()

		if c <= 10 {

			if err != nil {
				log.Print(err)

				continue
			}

			fmt.Println(len(clients))

			go handle(conn)

		} else {
			conn.Write([]byte("Chat is full. Try again later!"))
			fmt.Println("COUNT OF CLIENTS: ", countOfClients)
			mutex.Lock()
			countOfClients--
			mutex.Unlock()
			conn.Close()
		}

	}
}

func handle(conn net.Conn) {
	file, err := os.Open(filename) // 1 WELCOME
	defer file.Close()

	ErrorHandler(err)

	r := bufio.NewReader(file)
	buf := make([]byte, 500)
	io.ReadFull(r, buf)

	// ErrorHandler(err)

	_, err = conn.Write(buf)

	ErrorHandler(err)

	user, err := userExist(conn) // 2 ENTER USERNAME

	ErrorHandler(err)

	/*history, err := ioutil.ReadFile(filehistory.Name())
	if err != nil {
		fmt.Println("OSYNDA")
		log.Fatal(err)
	}
	conn.Write(history)*/
	mutex.Lock()
	conn.Write([]byte(historyOfMessages))
	mutex.Unlock()

	messages <- newMessage(user, " has joined our chat...", conn)

	input := bufio.NewScanner(conn)
	for input.Scan() {
		if checkCorrectEnter(input.Text()) {
			messages <- newMessage(user, input.Text(), conn)
		} else {
			conn.Write([]byte("Please enter latin or kirill letters! :"))
		}
	}

	// Delete client form map
	mutex.Lock()
	delete(clients, user)
	countOfClients--
	// fmt.Println("CLIENT US LEAVE COUNT: ", countOfClients)
	mutex.Unlock()

	fmt.Println(user+" LEFT len CLIENTS: ", len(clients))
	leaving <- newMessage(user, " has left our chat...", conn)

	conn.Close()
}

func newMessage(user string, msg string, conn net.Conn) message {
	addr := conn.RemoteAddr().String()
	msgTime := "[" + time.Now().Format("01-02-2006 15:04:05") + "]"
	if msg != "" && msg != " has left our chat..." && msg != " has joined our chat..." {
		mutex.Lock()
		historyOfMessages += msgTime + "[" + user + "]" + msg + "\n"
		mutex.Unlock()
		/*f, err := os.OpenFile(filehistory.Name(), os.O_APPEND|os.O_WRONLY, 0600)
		defer f.Close()
		ErrorHandler(err)
		if _, err = f.WriteString(msgTime + "[" + user + "]" + msg + "\n"); err != nil {
			panic(err)
		}*/
	}
	return message{
		time:     msgTime,
		userName: user,
		text:     msg,
		address:  addr,
	}
}

func userExist(conn net.Conn) (string, error) {
	scanner := bufio.NewScanner(conn)

	var name string

	for scanner.Scan() {
		name = strings.TrimSpace(scanner.Text())

		if !checkCorrectEnter(name) {
			conn.Write([]byte("Please enter latin or kirill letters! :"))
			continue
		}

		mutex.Lock()
		_, checkName := clients[name]
		mutex.Unlock()

		if checkName || len(name) == 0 {
			_, err := conn.Write([]byte("Enter the correct name or this Username already exists\n" + "[ENTER YOUR NAME]: "))
			if ErrorHandler(err) != nil {
				return "", err
			}
		} else {
			mutex.Lock()
			clients[name] = conn

			// fmt.Println("NEW CONNECTION: ", countOfClients)
			mutex.Unlock()
			break
		}
	}

	return name, nil
}

func ErrorHandler(err error) error {
	if err != nil {
		log.Println(err)
	}
	return err
}

func broadcaster() {
	for {
		select {
		case msg := <-messages:
			mutex.Lock()
			for nameClient, conn := range clients {

				if msg.address == conn.RemoteAddr().String() {
					if msg.text == " has joined our chat..." { // waiting load history
					}
					fmt.Fprint(conn, msg.time+"["+msg.userName+"]"+":")
					continue
				}
				if msg.text == " has joined our chat..." {

					clients[nameClient].Write([]byte(ClearLine(msg.text) + msg.userName + msg.text + "\n"))

					fmt.Fprint(conn, msg.time+"["+nameClient+"]"+":")
					continue
				}
				if msg.text != "" {
					fmt.Fprintln(conn, ClearLine(msg.text)+msg.time+"["+msg.userName+"]"+":"+msg.text)
					fmt.Fprint(conn, msg.time+"["+nameClient+"]"+":")
				}

			}
			mutex.Unlock()

			if msg.text == " has joined our chat..." {
				mutex.Lock()
				historyOfMessages += msg.userName + msg.text + "\n"
				mutex.Unlock()
			}
		case msg := <-leaving:

			historyOfMessages += msg.userName + msg.text + "\n"
			mutex.Lock()
			for nameClient, conn := range clients {
				clients[nameClient].Write([]byte(ClearLine(msg.text) + msg.userName + msg.text + "\n"))

				fmt.Fprint(conn, msg.time+"["+nameClient+"]:") // NOTE: ignoring network errors

			}
			mutex.Unlock()

		}
	}
}

/*
func ErrorHandler(err error) {
	if err != nil {
		log.Println(err)
	}
}
*/

func ClearLine(s string) string {
	return "\r" + strings.Repeat(" ", len(s)+len(s)/2) + "\r"
}

func checkCorrectEnter(s string) bool {
	if s == "" {
		return false
	}
	for _, v := range s {
		if v < 32 {
			return false
		}
	}
	return true
}
