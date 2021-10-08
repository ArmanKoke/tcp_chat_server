package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func typeCommand() {
	time.Sleep(100 * time.Millisecond) // Need improvement with mutex
	fmt.Print("Type command: ")
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Specify port in arguments!")
	}
	port := ":" + os.Args[1]
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Println("Server started on port:", port)

	core := NewCore()
	go core.Run()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		client := NewClient(
			conn,
			core.orders,
			core.register,
			core.unregister,
		)
		go client.Read()
	}
}

type Message struct {
	Len     int
	Content []byte
	Str     string
}

// Core
type ID int

const (
	REG ID = iota
	MSG
	ALL
	CLIENTS
)

type Command struct {
	id        ID
	recipient string
	sender    string
	msg       Message
}

type Core struct {
	Clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	orders     chan Command
}

func NewCore() *Core {
	return &Core{
		Clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		orders:     make(chan Command),
	}
}

func (c *Core) Run() {
	for {
		go c.Read()

		select {
		case client := <-c.register:
			c.Register(client)
		case client := <-c.unregister:
			c.Unregister(client)
		case cmd := <-c.orders:
			switch cmd.id {
			case MSG:
				c.message(cmd.sender, cmd.recipient, cmd.msg)
			}
		}
	}
}

func (c *Core) Read() error {
	for {
		go typeCommand()
		msg, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		msg = msg[:len(msg)-1]
		cmd := bytes.TrimSpace(bytes.Split(msg, []byte(" "))[0])
		args := bytes.TrimSpace(bytes.TrimPrefix(msg, cmd))

		// For different scopes simplified
		switch string(cmd) {
		case "CLIENTS":
			c.listClients()
		case "ALL":
			c.broadcast(args)
		case "":
			continue
		default:
			fmt.Println("Command not found")
		}
	}
}

func (c *Core) Register(client *Client) {
	fmt.Println("\nNew user: " + client.Name)
	c.Clients[client.Name] = client
	client.Write([]byte("Done!"))
}

func (c *Core) Unregister(client *Client) {
	fmt.Println("\nUser left: " + client.Name)
	delete(c.Clients, client.Name)
}

// For simplicity left println
func (c *Core) listClients() {
	if len(c.Clients) == 0 {
		fmt.Println("No active users!")
		return
	}
	clients := make([]string, 0)
	for name := range c.Clients {
		clients = append(clients, "@"+name)
	}
	fmt.Printf("Current active users: %v\n", strings.Join(clients, ", "))
}

func (c *Core) broadcast(msg []byte) {
	if len(c.Clients) == 0 {
		fmt.Println("No active users to broadcast!")
		return
	}
	for n, client := range c.Clients {
		fmt.Println("@"+n, string(msg))
		client.Write(msg)
	}
}

func (c *Core) message(sender, recipient string, msg Message) {
	if client, ok := c.Clients[recipient]; ok {
		client.Write(append([]byte("@"+sender+" "), msg.Content...))
	}
}

// Client
type Client struct {
	Conn       net.Conn
	Name       string
	Order      chan<- Command
	Register   chan<- *Client
	Unregister chan<- *Client
}

func NewClient(c net.Conn, order chan<- Command, reg chan<- *Client, unreg chan<- *Client) *Client {
	return &Client{
		Conn:       c,
		Order:      order,
		Register:   reg,
		Unregister: unreg,
	}
}

func (c *Client) Read() error {
	for {
		msg, err := bufio.NewReader(c.Conn).ReadBytes('\n')
		if err == io.EOF {
			c.Unregister <- c
			return nil
		}
		if err != nil {
			return err
		}

		c.handle(msg)
	}
}

func (c *Client) handle(msg []byte) {
	cmd := bytes.TrimSpace(bytes.Split(msg, []byte(" "))[0])
	args := bytes.TrimSpace(bytes.TrimPrefix(msg, cmd))
	fmt.Printf("\nUser @%s command: %s %s\n", c.Name, cmd, args)

	switch string(cmd) {
	case "REG":
		if err := c.register(args); err != nil {
			c.err(err)
		}
	case "MSG":
		if err := c.msg(args); err != nil {
			c.err(err)
		}
	default:
		c.err(fmt.Errorf("unknown command %s", cmd))
	}
}

func (c *Client) err(stack error) {
	c.Write([]byte(fmt.Sprintf("Error: %s \n", stack.Error())))
}

func (c *Client) register(args []byte) error {
	u := bytes.TrimSpace(args)
	if len(u) == 0 {
		return fmt.Errorf("nickname cannot be blank")
	}

	c.Name = string(u)
	c.Register <- c
	return nil
}

var Splitter = []byte("//")

func (c *Client) msg(args []byte) error {
	args = bytes.TrimSpace(args)
	if args[0] != '@' {
		return fmt.Errorf("recipient must be user ('@nickname')")
	}

	recipient := bytes.Split(args, []byte(" "))[0]
	if len(recipient) == 0 {
		return fmt.Errorf("recipient must have a nickname")
	}

	args = bytes.TrimSpace(bytes.TrimPrefix(args, recipient)) // 4//fold
	bodyLen := bytes.Split(args, Splitter)[0]                 // 4
	length, err := strconv.Atoi(string(bodyLen))
	if err != nil {
		return fmt.Errorf("no message body")
	}
	if length == 0 {
		return fmt.Errorf("message body is empty")
	}

	padding := len(bodyLen) + len(Splitter) //3 4//
	body := args[padding : padding+length]  //fold

	c.Order <- Command{
		recipient: string(recipient[1:]),
		sender:    c.Name,
		msg: Message{
			Len:     int(length),
			Content: body,
			Str:     string(body),
		},
		id: MSG,
	}

	return nil
}

func (c *Client) Write(b []byte) {
	if _, err := c.Conn.Write(append(b, []byte("\n")...)); err != nil {
		fmt.Println("Error while writing to client:", err)
	}
}
