package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/uebian/wsudp/config"
	"github.com/uebian/wsudp/service"
)

func main() {
	cfg := config.New()
	cfg.Load()
	if cfg.Mode == "server" {
		server := service.NewServer(cfg)
		err := server.Init()
		if err != nil {
			log.Fatal("Failed to start Server:", err)
		}
		defer server.Close()

		go server.ListenAndServe()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		log.Println("Shutdown Server ...")

		if err = server.Close(); err != nil {
			log.Fatal("Server Shutdown:", err)
		}
		log.Println("Server exiting")

	} else if cfg.Mode == "client" {
		client := service.NewClient(cfg)
		err := client.Init()
		if err != nil {
			log.Fatal("Failed to start client:", err)
		}

		defer client.Close()

		go client.ListenAndServe()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		log.Println("Shutdown Client ...")

		if err := client.Close(); err != nil {
			log.Fatal("Client Shutdown:", err)
		}
		log.Println("Client exiting")

	} else {
		log.Fatalf("Unknown mode %s", cfg.Mode)
	}

}
