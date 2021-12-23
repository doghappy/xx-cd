package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var port = flag.Int("p", 9598, "端口")
var py = flag.String("py", "C:\\Python27\\python.exe", "Python路径")

const (
	Log = iota
	Register
	Build
)

var clients = make(map[string]*textproto.Conn)
var reglock sync.RWMutex
var buildingLock sync.Mutex
var buildingFor string
var buildingAt time.Time
var sema = make(chan struct{}, 20)

func main() {
	flag.Parse()
	address := ":" + strconv.Itoa(*port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		c := textproto.NewConn(conn)
		go handleConn(c)
	}
}

func handleConn(conn *textproto.Conn) {
	for {
		text, err := conn.ReadLine()
		if err != nil {
			log.Println(err)
			break
		}
		items := strings.Split(text, ";")
		protocol, err := strconv.Atoi(items[0])
		if err != nil {
			log.Println(err)
			continue
		}
		switch protocol {
		case Register:
			handleRegister(conn, items[1:])
		case Build:
			handleBuild(conn, items[1:])
		default:
			conn.Cmd("0;Invalid protocol")
			conn.Close()
		}
	}

	key, ok := getUser(conn)
	if ok {
		reglock.Lock()
		delete(clients, key)
		reglock.Unlock()
		log.Println(key + "已断开")
	}
}

func handleRegister(conn *textproto.Conn, items []string) {
	if len(items) == 0 {
		conn.Cmd("5;Invalid parameters")
		conn.Close()
		return
	}
	user := items[0]
	reglock.RLock()
	_, ok := clients[user]
	reglock.RUnlock()
	if ok {
		conn.Cmd("5;User already exists")
		conn.Close()
	} else {
		reglock.Lock()
		clients[user] = conn
		reglock.Unlock()
		msg := fmt.Sprintf("%s 已连接", user)
		log.Println(msg)
		conn.Cmd("1;%s", msg)
	}
}

func handleBuild(conn *textproto.Conn, items []string) {
	if len(items) == 0 {
		conn.Cmd("5;Invalid parameters")
		conn.Close()
		return
	}
	dir := items[0]
	reglock.RLock()
	user, ok := getUser(conn)
	reglock.RUnlock()
	if ok {
		if buildingFor != "" {
			conn.Cmd("0;正在为 %s 编译，编译开始于 %s ", buildingFor, buildingAt.Format("2006-01-02 15:04:05"))
			conn.Cmd("0;排队中...")
		}
		buildingLock.Lock()
		quit := make(chan struct{})
		tick := time.Tick(5 * time.Second)
		go func() {
			for {
				select {
				case <-quit:
					return
				case <-tick:
					conn.Cmd("0;正在编译中...")
				}
			}
		}()
		buildingAt = time.Now()
		buildingFor = user
		src := path.Join("config", dir)
		dest := "project/20191223-base/config"
		errch := make(chan error)
		var wg sync.WaitGroup
		wg.Add(1)
		go sendError(conn, errch)
		go copyDir(src, dest, &wg, errch)
		wg.Wait()
		close(errch)

		build(conn, user)

		buildingFor = ""
		quit <- struct{}{}
		buildingLock.Unlock()
		conn.Cmd("2;编译结束")
	} else {

	}
}

func sendError(conn *textproto.Conn, ch <-chan error) {
	for err := range ch {
		conn.Cmd("4;%s", err.Error())
	}
}

func copyDir(src string, dest string, wg *sync.WaitGroup, ch chan<- error) {
	sema <- struct{}{}
	defer func() {
		<-sema
		wg.Done()
	}()
	files, err := ioutil.ReadDir(src)
	if err != nil {
		log.Println(err)
		ch <- err
	}
	for _, file := range files {
		name := file.Name()
		if file.IsDir() {
			wg.Add(1)
			newsrc := path.Join(src, name)
			newdest := path.Join(dest, name)
			go copyDir(newsrc, newdest, wg, ch)
		} else {
			s := path.Join(src, name)
			d := path.Join(dest, name)
			data, err := ioutil.ReadFile(s)
			if err != nil {
				log.Println(err)
				ch <- err
			}
			err = ioutil.WriteFile(d, data, 0644)
			if err != nil {
				log.Println(err)
				ch <- err
			}
		}
	}
}

func getUser(conn *textproto.Conn) (key string, ok bool) {
	for k, v := range clients {
		if v == conn {
			key = k
			ok = true
			return
		}
	}
	return
}

func build(conn *textproto.Conn, user string) {
	cmd := exec.Command(*py, "zipConfigJson.py")
	cmd.Dir = "project/20191223-base/"
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
	if err != nil {
		log.Println(err)
		conn.Cmd("4;%s", err.Error())
		return
	}

	name := "D:\\CocosDashboard_1.0.20\\resources\\.editors\\Creator\\2.4.3\\CocosCreator.exe"
	argv := []string{
		"--path",
		"D:\\build\\project\\20191223-base",
		"--build",
		fmt.Sprintf("platform=web-desktop;debug=true;previewWidth=1080;previewHeight=1920;md5Cache=true;buildPath=D:\\WebSites\\WebClient\\%s", user),
	}
	procAttr := &os.ProcAttr{
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
	}
	p, err := os.StartProcess(name, argv, procAttr)
	if err != nil {
		log.Println(err)
		conn.Cmd("4;%s", err.Error())
		//time.Sleep(20 * time.Second)
		return
	}
	state, err := p.Wait()
	if err != nil {
		log.Println(err)
		conn.Cmd("4;%s", err.Error())
		return
	}
	err = p.Release()
	if err != nil {
		log.Println(err)
		conn.Cmd("4;%s", err.Error())
		return
	}
	conn.Cmd("2;ExitCode %d", state.ExitCode())
}
