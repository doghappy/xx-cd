package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

var host = flag.String("h", "192.168.2.230:9598", "服务端套接字地址")
var user string

func register() {
	log.Println("请输入名字：")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	user = input.Text()
}

func main() {
	flag.Parse()

	register()

	c, err := net.Dial("tcp", *host)
	if err != nil {
		log.Fatalln(err)
	}
	conn := textproto.NewConn(c)
	defer conn.Close()

	conn.Cmd("1;%s", user)
	text, err := conn.ReadLine()
	if err != nil {
		log.Fatalln(err)
	}

	if p := handle(text); p == 5 {
		return
	}

	for {
		log.Println("请输入操作  b: 编译  q: 退出")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		text = input.Text()
		switch text {
		case "q":
			conn.Close()
			return
		case "b":
			conn.Cmd("2;%s", user)
			for {
				text, err := conn.ReadLine()
				if err != nil {
					log.Println(err)
				}
				if text == "" {
					log.Println("连接已断开")
					return
				}
				if p := handle(text); p == 2 {
					break
				}
			}
		}
	}
}

func handle(msg string) int {
	items := strings.Split(msg, ";")
	protocol, err := strconv.Atoi(items[0])
	if err != nil {
		log.Println(msg)
		log.Println(err)
		return -1
	}
	switch protocol {
	case 4:
		log.Printf("错误：%s\n", items[1:][0])
	default:
		log.Println(items[1:][0])
	}
	return protocol
}
