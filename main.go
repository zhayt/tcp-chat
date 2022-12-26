package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	clients  = make(map[string]net.Conn)
	leaving  = make(chan message)
	messages = make(chan message)
)

type message struct {
	text    string
	address string
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
	c.Write([]byte("Welcome to the TCP chat!\nEnter your name: "))
	for {
		c.Write([]byte("Enter your name: "))
		name, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			log.Print(err)
			continue
		}
		if !nameChecker(name) {
			c.Write([]byte("bad name"))
			continue
		}
		clients[name] = c
		break
	}
	go msgSwitch()
	reader := bufio.NewScanner(c)
	newMsg := message{}
	for reader.Scan() {
		messages <- newMsg.Add(reader.Text(), c)
	}

	delete(clients, c.RemoteAddr().String())

	leaving <- newMsg.Add(" has left", c)
	c.Close()
}

func (t *message) Add(text string, c net.Conn) message {
	return message{
		text:    c.RemoteAddr().String() + text,
		address: c.RemoteAddr().String(),
	}
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
			for _, c := range clients {
				if msg.address == c.RemoteAddr().String() {
					continue
				}
				c.Write([]byte(msg.text + "\n"))
			}
		case msg := <-leaving:
			for _, c := range clients {
				c.Write([]byte(msg.text + "\n"))
			}

		}

	}
}
