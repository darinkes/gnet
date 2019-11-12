package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/panjf2000/gnet"
)

type udpClient struct {
	*gnet.EventServer
}

var establishedChannel chan *gnet.Client

func (uc *udpClient) OnConnectionEstablished(c *gnet.Client) (action gnet.Action) {
	fmt.Printf("Established: %v -> %v\n", c.LocalAddr(), c.RemoteAddr())
	establishedChannel <- c
	return
}

func (uc *udpClient) React(c gnet.Conn) (out []byte, action gnet.Action) {
	fmt.Print(string(c.Read()))
	c.ResetBuffer()
	return
}

func main() {
	fmt.Println("UDP-Client")

	establishedChannel = make(chan *gnet.Client, 1)

	udpClient := new(udpClient)

	go func() {
		err := gnet.Connect(udpClient, "udp://127.0.0.1:5555")
		if err != nil {
			panic(err)
		}
	}()
	client := <-establishedChannel

	fmt.Print("Let's chat: ")
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		client.Write([]byte(text))
	}
}
