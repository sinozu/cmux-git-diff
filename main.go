package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

func main() {
	port := flag.Int("port", 8080, "server port")
	bind := flag.String("bind", "localhost", "bind address")
	interval := flag.Duration("interval", 3*time.Second, "polling interval")
	flag.Parse()

	if *bind != "localhost" && *bind != "127.0.0.1" {
		log.Printf("WARNING: binding to %s exposes diff content (possibly sensitive code) to the network", *bind)
	}

	repoDir, err := GetRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not a git repository")
		os.Exit(1)
	}

	repoName := GetRepoName(repoDir)
	addr := fmt.Sprintf("%s:%d", *bind, *port)
	url := fmt.Sprintf("http://%s", addr)

	isLocal := *bind == "localhost" || *bind == "127.0.0.1"
	srv := NewServer(repoName, isLocal)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}

	// Start watcher in background
	watcher := NewWatcher(repoDir, *interval)
	go watcher.Watch(func(result *DiffResult) {
		srv.UpdateDiff(result)
	})

	// Start HTTP server
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	go httpServer.Serve(ln)

	log.Printf("git-diff-browser: %s on %s", repoName, url)

	// Open in cmux browser pane if available
	openBrowser(url)

	// Wait for interrupt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down...")
	watcher.Stop()
	httpServer.Shutdown(context.Background())
}

func openBrowser(url string) {
	wsID := os.Getenv("CMUX_WORKSPACE_ID")
	if wsID != "" {
		cmux, err := exec.LookPath("cmux")
		if err == nil {
			cmd := exec.Command(cmux, "new-pane",
				"--type", "browser",
				"--workspace", wsID,
				"--url", url,
			)
			if err := cmd.Run(); err != nil {
				log.Printf("cmux new-pane: %v", err)
			}
			return
		}
	}

	// Fallback: just print the URL
	fmt.Printf("\n  Open in browser: %s\n\n", url)
}
