package main

import (
	"bitbucket.org/gdamore/mangos"
	"bitbucket.org/gdamore/mangos/protocol/pub"
	"bitbucket.org/gdamore/mangos/protocol/sub"
	"bitbucket.org/gdamore/mangos/transport/tcp"
	"flag"
	"fmt"
	"github.com/dmarkham/pidfile"
	"github.com/pelletier/go-toml"
	"os"
	"runtime"
	"syscall"
)

var (
	pubEndpoint, subEndpoint, pidFilePath string
	maxProcs                              int
	daemonize                             *bool = flag.Bool("d", false, "daemonize options")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Other Options:
  GO_ENV=development: Set environments
`)
}

func init() {
	var confFilePath *string = flag.String("c", "./config.toml", "config file path")
	flag.Usage = Usage
	flag.Parse()

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}
	fmt.Println("Loading", env, "environment")

	fmt.Println("Loading the config file:", *confFilePath)
	tree, err := toml.LoadFile(*confFilePath)
	if err != nil {
		fmt.Println("Got an unexpected error reading file:", err)
	}

	pubEndpoint = tree.Get(env + ".pub_endpoint").(string)
	if pubEndpoint == "" {
		pubEndpoint = "tcp://127.0.0.1:19018"
	}
	fmt.Println("pubEndpoint", pubEndpoint)

	subEndpoint = tree.Get(env + ".sub_endpoint").(string)
	if subEndpoint == "" {
		subEndpoint = "tcp://127.0.0.1:19019"
	}
	fmt.Println("subEndpoint", subEndpoint)

	pidFilePath = tree.Get(env + ".pidfile").(string)
	if pidFilePath == "" {
		pidFilePath = "/tmp/nanomsg.pid"
	}
	fmt.Println("pidfile", pidFilePath)

	maxprocs := tree.Get(env + ".maxprocs")
	if maxprocs != nil {
		maxProcs = int(maxprocs.(int64))
		if maxProcs < 1 {
			maxProcs = 1
		}
	} else {
		maxProcs = runtime.NumCPU()
	}
	fmt.Println("GOMAXPROCS:", maxProcs)
	runtime.GOMAXPROCS(maxProcs)
}

func daemon(nochdir, noclose int) int {
	var ret uintptr
	var err syscall.Errno

	ret, _, err = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return -1
	}
	switch ret {
	case 0:
		pidfile.SetPidfile(pidFilePath)
		break
	default:
		os.Exit(0)
	}

	pidFilePath, _ := syscall.Setsid()
	if pidFilePath == -1 {
		return -1
	}

	if nochdir == 0 {
		os.Chdir("/")
	}

	syscall.Umask(0)

	if noclose == 0 {
		f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if e == nil {
			fd := int(f.Fd())
			syscall.Dup2(fd, int(os.Stdin.Fd()))
			syscall.Dup2(fd, int(os.Stdout.Fd()))
			syscall.Dup2(fd, int(os.Stderr.Fd()))
		}
	}
	return 0
}

func newPublisher() (socket mangos.Socket, err error) {
	socket, err = pub.NewSocket()
	if err != nil {
		return
	}
	socket.AddTransport(tcp.NewTransport())
	return
}

func newSubscriber() (socket mangos.Socket, err error) {
	socket, err = sub.NewSocket()
	if err != nil {
		return
	}
	socket.AddTransport(tcp.NewTransport())
	socket.SetOption(mangos.OptionSubscribe, []byte(""))
	return
}

func main() {
	if *daemonize == true {
		errcode := daemon(0, 0)
		if errcode != 0 {
			fmt.Println("daemon err!!")
			os.Exit(1)
		}
	}
	path, _ := os.Getwd()
	os.Chdir(path)

	publisher, err := newPublisher()
	if err != nil {
		fmt.Println("NewSocket(): %v", err)
		os.Exit(0)
	}
	defer publisher.Close()
	err = publisher.Listen(pubEndpoint)
	if err != nil {
		fmt.Println("publisher.Listen(): %v", err)
		os.Exit(0)
	}

	subscriber, err := newSubscriber()
	if err != nil {
		fmt.Println("newSubscriber(): %v", err)
		os.Exit(0)
	}
	defer subscriber.Close()
	err = subscriber.Listen(subEndpoint)
	if err != nil {
		fmt.Println("subscriber.Listen(): %v", err)
		os.Exit(0)
	}

	for {
		msg, err := subscriber.Recv()
		if err == nil {
			_ = publisher.Send(msg)
		}
	}
}
