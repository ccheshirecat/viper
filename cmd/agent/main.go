package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ccheshirecat/viper/internal/agent"
)

func main() {
	var (
		listen  = flag.String("listen", ":8080", "HTTP listen address")
		vmName  = flag.String("vm-name", "", "VM name identifier")
		taskDir = flag.String("task-dir", "/var/viper/tasks", "Task storage directory")
	)
	flag.Parse()

	if *vmName == "" {
		log.Fatal("vm-name is required")
	}

	server, err := agent.NewServer(*listen, *vmName, *taskDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		log.Printf("Agent starting for VM %s on %s", *vmName, *listen)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down agent...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
