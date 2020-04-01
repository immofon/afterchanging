package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dietsche/rfsnotify"
	"gopkg.in/fsnotify.v1"
)

type Info struct {
	Size    int64
	ModTime time.Time
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("help: afterchanging name [args]")
		return
	}

	var cmdName string = os.Args[1]
	var cmdArgs []string
	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}

	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	//rfsnotify adds two new API entry points:
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	watcher.AddRecursive(pwd)

	restart := make(chan string, 16)
	restart <- ""

	infos := make(map[string]*Info)
	var (
		ctx    context.Context
		cancel context.CancelFunc = func() {}
	)
	go func() {
		for name := range restart {
			stat, err := os.Stat(name)
			if err == nil {
				info := infos[name]
				if info == nil {
					info = &Info{}
				}

				if info.Size == stat.Size() &&
					info.ModTime == stat.ModTime() {
					continue
				}

				info.Size = stat.Size()
				info.ModTime = stat.ModTime()

			}
			delete(infos, name)

			cancel()

			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			go cmd.Run()
		}
	}()

	for {
		select {
		case e := <-watcher.Events:
			if e.Op == fsnotify.Chmod {
				continue
			}
			if strings.HasSuffix(e.Name, ".go") {
				restart <- e.Name
			}
		}
	}
}
