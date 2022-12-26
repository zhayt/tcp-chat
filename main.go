package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
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

func (t *message) Add(text string, c net.Conn) *message {
	return &message{
		text:    c.RemoteAddr().String() + text,
		address: c.RemoteAddr().String(),
	}
}

func (t *message) Check() bool {
	fmt.Println("HERO")
	fmt.Println(t.text)
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

func main() {
	l, err := net.Listen("tcp", ":9001")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Print(err)
			os.Exit(100)
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	c.Write([]byte("Welcome to the TCP chat!\n"))
	var userName string
	newMsg := message{}
	for {
		c.Write([]byte("Enter your name: "))
		name, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			log.Print(err)
			continue
		}
		if !nameChecker(name) {
			c.Write([]byte("bad name, try again!\n"))
			continue
		}
		userName = strings.TrimSuffix(name, "\n")
		clients[userName] = c
		break
	}
	go msgSwitch()
	reader := bufio.NewScanner(c)
	c.Write([]byte(fmt.Sprintf("\r[%s][%s]:", time.Now().Format("01-02-2022 15:04:05"), userName)))
	for reader.Scan() {
		c.Write([]byte(fmt.Sprintf("\r[%s][%s]:", time.Now().Format("01-02-2022 15:04:05"), userName)))
		if !newMsg.Add(strings.TrimSpace(reader.Text()), c).Check() {
			prefix := fmt.Sprintf("\r[%s][%s]:", time.Now().Format("01-02-2022 15:04:05"), userName)
			messages <- *newMsg.Add(prefix+strings.TrimSpace(reader.Text()), c)
		}
	}

	delete(clients, c.RemoteAddr().String())

	leaving <- *newMsg.Add(" has left our chat...", c)
	c.Close()
}

func nameChecker(name string) bool {
	fmt.Println(name)
	if len(name) < 4 || len(name) > 12 {
		return false
	}
	for _, v := range name {
		if v >= 'A' && v <= 'Z' || v >= 'a' && v <= 'z' || v == '\n' {
			continue
		}
		return false
	}
	return true
}

func msgSwitch() {
	for {
		select {
		case msg := <-messages:
			mu.Lock()
			for _, c := range clients {
				if msg.address == c.RemoteAddr().String() {
					continue
				}
				c.Write([]byte(msg.text + "\n"))
			}
			mu.Unlock()
		case msg := <-leaving:
			for _, c := range clients {
				c.Write([]byte(msg.text + "\n"))
			}
		}
	}
}
