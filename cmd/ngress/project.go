package main

import (
	"flag"
	"fmt"
	"github.com/petermattis/goid"
	"k8s.io/klog/v2"
	"os"
	"os/signal"
	"syscall"
)

var Version = "0.0.0"
var Hash = "xxxxxxxx"
var BuildDate = ""
var BuildAgent = ""

func buildInfo() string {
	return fmt.Sprintf("version: %v, build agent: '%v', build date: %v, hash: %v", Version, BuildAgent, BuildDate, Hash)
}

type Project struct {
	tag    string
	chStop chan os.Signal
}

type SLogFilter struct {
}

func (c *SLogFilter) Filter(args []interface{}) []interface{} {
	return args
}

func (c *SLogFilter) FilterF(format string, args []interface{}) (string, []interface{}) {
	return fmt.Sprintf("[%v] %v", goid.Get(), format), args
}

func (c *SLogFilter) FilterS(msg string, keysAndValues []interface{}) (string, []interface{}) {
	return msg, keysAndValues
}

func NewProject(tag string) *Project {
	klog.InitFlags(nil)
	defer klog.Flush()
	klog.SetLogFilter(&SLogFilter{})

	ver := flag.Bool("version", false, "get version")
	info := flag.Bool("info", false, "get build info")
	date := flag.Bool("date", false, "get build date")
	hash := flag.Bool("hash", false, "get hash")
	flag.Parse()

	if *ver {
		fmt.Println(Version)
		syscall.Exit(0)
	} else if *hash {
		fmt.Println(Hash)
		syscall.Exit(0)
	} else if *date {
		fmt.Println(BuildDate)
		syscall.Exit(0)
	} else if *info {
		fmt.Println(buildInfo())
		syscall.Exit(0)
	}
	klog.Infof("%v> START %v", tag, buildInfo())

	c := &Project{tag: tag, chStop: make(chan os.Signal, 2)}
	signal.Notify(c.chStop, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return c
}

func (c *Project) Close() {
	klog.Infof("%v> Closed succeffuly\n", c.tag)
}

func (c *Project) WaitStopSignal() syscall.Signal {
	sig := <-c.chStop
	switch v := sig.(type) {
	case syscall.Signal:
		klog.Infof("%v> STOP: syscall.Signal: %d\n", c.tag, v)
		return v
	default:
		klog.Infof("%v> STOP: default: %v\n", c.tag, v)
		return syscall.SIGQUIT
	}
}
