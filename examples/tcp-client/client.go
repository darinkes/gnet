package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/panjf2000/gnet"
)

type tcpClient struct {
	*gnet.EventServer
}

var establishedChannel chan *gnet.Client

func (uc *tcpClient) OnConnectionEstablished(c *gnet.Client) (action gnet.Action) {
	fmt.Printf("Established: %v -> %v\n", c.LocalAddr(), c.RemoteAddr())
	establishedChannel <- c
	return
}

func (uc *tcpClient) React(c gnet.Conn) (out []byte, action gnet.Action) {
	fmt.Print(string(c.Read()))
	c.ResetBuffer()
	return
}

func main() {
	fmt.Println("TCP-Client")

	establishedChannel = make(chan *gnet.Client, 1)

	tcpClient := new(tcpClient)

	go func() {
		err := gnet.Connect(tcpClient, "tcp://127.0.0.1:5555")
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
