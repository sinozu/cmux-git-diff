package main

import (
	"context"
	"encoding/json"
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
	port := flag.Int("port", 0, "server port (0 = auto-assign)")
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

	// Listen first, then resolve the actual port (supports port 0 = OS auto-assign)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *bind, *port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	actualAddr := ln.Addr().String()
	url := fmt.Sprintf("http://%s", actualAddr)

	isLocal := *bind == "localhost" || *bind == "127.0.0.1"
	srv := NewServer(repoName, isLocal)
	httpServer := &http.Server{
		Handler: srv.Handler(),
	}

	// Start watcher in background
	watcher := NewWatcher(repoDir, *interval)
	go watcher.Watch(func(result *DiffResult) {
		srv.UpdateDiff(result)
	})

	go httpServer.Serve(ln)

	log.Printf("cmux-git-diff: %s on %s", repoName, url)

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
	if wsID == "" {
		fmt.Printf("\n  Open in browser: %s\n\n", url)
		return
	}

	cmuxBin, err := exec.LookPath("cmux")
	if err != nil {
		fmt.Printf("\n  Open in browser: %s\n\n", url)
		return
	}

	// Get current pane from cmux identify
	paneRef := getCmuxPaneRef(cmuxBin, wsID)
	if paneRef != "" {
		// Open as a browser tab in the same pane
		cmd := exec.Command(cmuxBin, "new-surface",
			"--type", "browser",
			"--pane", paneRef,
			"--workspace", wsID,
			"--url", url,
		)
		if err := cmd.Run(); err != nil {
			log.Printf("cmux new-surface: %v", err)
		}
		return
	}

	// Fallback: new pane
	cmd := exec.Command(cmuxBin, "new-pane",
		"--type", "browser",
		"--workspace", wsID,
		"--url", url,
	)
	if err := cmd.Run(); err != nil {
		log.Printf("cmux new-pane: %v", err)
	}
}

func getCmuxPaneRef(cmuxBin, wsID string) string {
	surfaceID := os.Getenv("CMUX_SURFACE_ID")
	if surfaceID == "" {
		return ""
	}

	out, err := exec.Command(cmuxBin, "identify",
		"--surface", surfaceID,
		"--workspace", wsID,
	).Output()
	if err != nil {
		return ""
	}

	var result struct {
		Caller struct {
			PaneRef string `json:"pane_ref"`
		} `json:"caller"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return ""
	}
	return result.Caller.PaneRef
}
