package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func startGoTest(args ...string) error {
	cmd := exec.Command("go", append([]string{"test"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting go test: %s", err)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("error running go test: %s", err)
	}
	return nil
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("failed to get working directory:", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("failed to initialize fsnotify:", err)
	}

	err = watcher.Watch(wd)
	if err != nil {
		log.Fatal("failed to watch working directory:", err)
	}
	defer watcher.Close()

	events := make(map[string]*fsnotify.FileEvent)
	debounce := make(<-chan time.Time)
	log.Println("started goautotest for", wd)

	// Run an initial test on startup
	if err := startGoTest(os.Args[1:]...); err != nil {
		log.Println(err)
	}

	for {
		select {
		case ev := <-watcher.Event:
			name, err := filepath.Rel(wd, ev.Name)
			if err != nil {
				name = ev.Name
			}
			events[name] = ev
			debounce = time.After(2 * time.Second)
		case err := <-watcher.Error:
			log.Println("watcher error:", err)
		case <-debounce:
			var runTests bool
			for k, v := range events {
				if strings.HasSuffix(k, ".go") {
					log.Println(eventDesc(v), k)
					runTests = true
				}
			}
			if runTests {
				err := startGoTest(os.Args[1:]...)
				if err != nil {
					log.Println(err)
				}
			}
			events = make(map[string]*fsnotify.FileEvent)
		}
	}
}

func eventDesc(ev *fsnotify.FileEvent) string {
	switch {
	case ev.IsCreate():
		return "created"
	case ev.IsDelete():
		return "removed"
	case ev.IsModify():
		return "modified"
	case ev.IsRename():
		return "renamed"
	default:
		return ""
	}
}
