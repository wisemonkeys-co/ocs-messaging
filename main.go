package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
)

func startSubMode(nc *nats.Conn) {
	fmt.Println("Starting sub mode")
	_, subError := nc.Subscribe("foo", func(m *nats.Msg) {
		fmt.Printf("Received a message: %s\n", string(m.Data))
	})
	if subError != nil {
		fmt.Println(subError)
	}
	select {}
}

func startPubMode(nc *nats.Conn) {
	fmt.Println("Starting pub mode")
	reader := bufio.NewReader(os.Stdin)
	for {
    fmt.Print("Enter text: ")
    text, readConsoleError := reader.ReadString('\n')
		if readConsoleError != nil {
			panic(readConsoleError)
		}
		pubError := nc.Publish("foo", []byte(strings.TrimRight(text, "\n")))
		if pubError != nil {
			panic(pubError)
		}
	}
}

func main ()  {
	nc, connectError := nats.Connect(nats.DefaultURL)
	if connectError != nil {
		fmt.Println(connectError)
		return;
	}
	defer nc.Close()
	var mode string
	for _, flag := range os.Args {
		if flag == "pub" {
			mode = "pub"
			break
		}
	}
	if mode == "pub" {
		startPubMode(nc)
	} else {
		startSubMode(nc)
	}
}