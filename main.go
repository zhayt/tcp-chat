package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	clients  = make(map[string]net.Conn)
	leaving  = make(chan message)
	messages = make(chan message)
	mu       sync.Mutex
)

type message struct {
	text    string
	address string
}

func (t message) Add(text string, c net.Conn) *message {
	return &message{
		text:    text,
		address: c.RemoteAddr().String(),
	}
}

func (t message) Check() bool {
	if t.text == "" {
		return true
	}
	for _, val := range t.text {
		if val < ' ' || val > '~' {
			return true
		}
	}
	return false
}

func loger(s string, file os.File) {
	str := fmt.Sprintf("[%s]%s\n", time.Now().Format("01-02-2022 15:04:05"), s)
	file.Write([]byte(str))
	fmt.Print(str)
}

func main() {
	PORT := ":9000"
	arguments := os.Args
	if len(arguments) != 1 {
		newPort, err := strconv.Atoi(arguments[1])
		if err == nil {
			if newPort > 1023 && newPort < 49152 {
				PORT = ":" + strconv.Itoa(newPort)
			}
		}
	}
	l, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	f1, err := os.Create("history.txt")
	f2, err := os.Create("log.txt")
	loger(fmt.Sprintf("[SERVER WAS STARTED][PORT%v]", PORT), *f2)
	defer f1.Close()
	defer f2.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			log.Print(err)
			os.Exit(100)
		}
		loger(fmt.Sprintf("[CONNECT THE SERVER][ADDRESS:%v]", c.RemoteAddr().String()), *f2)
		go handleConnection(c, *f1, *f2)
	}
}

func handleConnection(c net.Conn, fileHistory, fileLog os.File) {
	defer fileHistory.Close()
	defer fileLog.Close()
	c.Write([]byte("Welcome to the TCP chat!\n"))
	logo, err := os.ReadFile("logo.txt")
	if err != nil {
		loger(fmt.Sprintf("[COUDN'T OPEN FILE <logo.txt>][ADDRESS:%v]"), fileLog)
		c.Write([]byte("Nefig bylo ydolyt' logo.txt!!!\nPinguine\n"))
	}
	c.Write(logo)
	var userName string
	newMsg := message{}
	for {
		c.Write([]byte("[ENTER YOUR NAME]: "))
		input, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			loger(fmt.Sprintf("[THE USER LOGGED OUT WITHOUT ENTERING A NAME][ADDRESS:%v][ERROR:%v]", c.RemoteAddr().String(), err), fileLog)
			break
		}
		userName, err = nameChecker(input)
		if err != nil {
			c.Write([]byte(err.Error()))
			continue
		}

		mu.Lock()
		//if _, ok := clients[userName]; ok {
		//	c.Write([]byte("Name already has! Please try again.\n"))
		//	continue
		//}
		//if len(clients) > 9 {
		//	c.Write([]byte("Sever already full! Please try later.\n"))
		//	continue
		//}
		clients[userName] = c
		mu.Unlock()
		loger(fmt.Sprintf("[CLIENT JOIN IN THE CHAT][USER NAME:%v][ADDRESS:%v]", userName, c.RemoteAddr().String()), fileLog)
		break
	}
	//messages <- *newMsg.Add(userName+"has joined our chat...", c)
	history, err := os.ReadFile(fileHistory.Name())
	if err != nil {
		loger(fmt.Sprintf("[COUDN'T READ HISTORY FILE][ADDRESS:%v][ERROR:%s]", c.RemoteAddr().String(), err.Error()), fileLog)
	}
	c.Write(history)
	go msgSwitch()

	//go func() {
	//	messages <- *newMsg.Add("\r"+userName+" has joined our chat..."+strings.Repeat(" ", 100000000), c)
	//}()
	//newStruct := message{text: "\r" + userName + " has joined our chat..." + strings.Repeat(" ", 10), address: c.RemoteAddr().String()}
	//messages <- *newMsg.Add("\r"+userName+" has joined our chat..."+strings.Repeat(" ", 10), c)
	newStruct := *newMsg.Add("\r"+strings.Repeat(" ", 100)+"\r"+userName+" has joined our chat...", c)
	messages <- newStruct
	reader := bufio.NewScanner(c)
	c.Write([]byte(template(userName)))

	for reader.Scan() {
		c.Write([]byte(template(userName)))
		if !newMsg.Add(strings.TrimSpace(reader.Text()), c).Check() {
			netDate := *newMsg.Add(template(userName)+strings.TrimSpace(reader.Text()), c)
			fileHistory.Write([]byte(netDate.text + "\n"))
			messages <- netDate
		}
	}

	loger(fmt.Sprintf("[CLIENT LEFT THE SERVER][USER NAME:%v][ADDRESS:%v]", userName, c.RemoteAddr().String()), fileLog)
	mu.Lock()
	delete(clients, userName)
	mu.Unlock()
	leaving <- *newMsg.Add("\n"+userName+" has left our chat...", c)
	c.Close()
}

func template(n string) string {
	return fmt.Sprintf("\r[%s][%s]:", time.Now().Format("01-02-2022 15:04:05"), n)
}

func nameChecker(str string) (string, error) {
	userName := strings.TrimSuffix(strings.TrimSpace(str), "\n")
	if len(userName) < 4 || len(userName) > 12 {
		return "", errors.New("Bad name! Len name must be more 2 and less 12 symbols\n")
	}
	for _, v := range userName {
		if v >= 'A' && v <= 'Z' || v >= 'a' && v <= 'z' {
			continue
		}
		return "", errors.New("Bad name! Name must has only latin alphabet\n")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := clients[userName]; ok {
		return "", errors.New("Name already has! Please try again.\n")
	}
	if len(clients) > 9 {
		return "", errors.New("Sever already full! Please try to connect later.\n")
	}
	return userName, nil
}

func msgSwitch() {
	for {
		select {
		case msg := <-messages:
			mu.Lock()
			for name, c := range clients {
				if msg.address == c.RemoteAddr().String() {
					c.Write([]byte(template(name)))
					continue
				}
				c.Write([]byte(msg.text + "\n"))
				c.Write([]byte(template(name)))
			}
			mu.Unlock()
		case msg := <-leaving:
			mu.Lock()
			for name, c := range clients {
				c.Write([]byte(msg.text + "\n"))
				c.Write([]byte(template(name)))
			}
			mu.Unlock()
		}
	}
}
