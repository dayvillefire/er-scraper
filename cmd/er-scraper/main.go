package main

import (
	"flag"
	"log"

	"github.com/dayvillefire/er-scraper/agent"
)

var (
	debug = flag.Bool("debug", false, "Debug")
	user  = flag.String("user", "", "ER username")
	pass  = flag.String("pass", "", "ER password")
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Printf("syntax: er-scraper [--flags] ACTION")
		return
	}

	if *user == "" {
		*user = agent.DEFAULT_USERNAME
	}
	if *pass == "" {
		*pass = agent.DEFAULT_PASSWORD
	}

	switch flag.Arg(0) {
	case "training":
		exportTraining()
	default:
		log.Printf("Valid actions: training")
		return
	}
}

func getAgent() *agent.Agent {
	return &agent.Agent{
		Debug:    *debug,
		LoginUrl: agent.DEFAULT_URL,
		Username: *user,
		Password: *pass,
	}
}
