package main

import (
	"flag"
	"log"
	"os"

	"github.com/dayvillefire/er-scraper/agent"
	"github.com/joho/godotenv"
)

var (
	debug      = flag.Bool("debug", false, "Debug")
	user, pass string
)

func main() {
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(flag.Args()) < 1 {
		log.Printf("syntax: er-scraper [--flags] ACTION")
		log.Printf("actions: events, training")
		return
	}

	user = os.Getenv("USERNAME")
	pass = os.Getenv("PASSWORD")

	switch flag.Arg(0) {
	case "events":
		exportEvents()
	case "training":
		exportTraining()
	default:
		log.Printf("Valid actions: events, training")
		return
	}
}

func getAgent() *agent.Agent {
	return &agent.Agent{
		Debug:    *debug,
		Username: user,
		Password: pass,
	}
}
