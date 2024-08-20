package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/dayvillefire/er-scraper/agent"
	"github.com/jbuchbinder/shims"
)

func exportCommon() *agent.Agent {
	a := getAgent()
	err := a.Init()
	if err != nil {
		panic(err)
	}
	return a
}

func exportEvents() {
	a := exportCommon()

	cal, err := a.ExportCalendar()
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("calendar.vcs", cal, 0644)
	if err != nil {
		panic(err)
	}
	log.Printf("INFO: Exported calendar.vcs")
}

func exportTraining() {
	a := exportCommon()

	log.Printf("INFO: Fetching all training class IDs")
	ids, full, err := a.GetAllTrainingClassIDs()
	if err != nil {
		panic(err)
	}

	exportTrainingFromData(a, ids, full)
}

func exportTrainingFromCSV(csvfile string) {
	a := exportCommon()

	csvdata, err := os.ReadFile(csvfile)
	if err != nil {
		panic(err)
	}

	log.Printf("INFO: Fetching all training class IDs")
	ids, full, err := a.GetAllTrainingClassIDsFromCSV(csvdata)
	if err != nil {
		panic(err)
	}

	exportTrainingFromData(a, ids, full)
}

func exportTrainingFromData(a *agent.Agent, ids []int, full [][]string) {
	cols := []string{
		"Class ID", "Name", "Class Date",
		"Length", "Category Name", "Station",
		"Evaluations", "Template", "Lead Instructor",
		"Instructors", "Resources", "Training Codes",
		"Location", "Objective", "Narrative",
	}

	lookupOut := []map[string]string{}
	for _, r := range full {
		item := map[string]string{}
		for k, v := range r {
			item[cols[k]] = v
		}
		lookupOut = append(lookupOut, item)
	}

	b, err := json.Marshal(lookupOut)
	if err != nil {
		log.Printf("ERR: %s", err.Error())
		panic(err)
	}

	os.MkdirAll(fmt.Sprintf("%s/training", shims.SingleValueDiscardError(os.Getwd())), 0755)
	err = os.WriteFile(fmt.Sprintf("%s/training/lookup.csv", shims.SingleValueDiscardError(os.Getwd())), b, 0644)
	if err != nil {
		log.Printf("ERR: %s", err.Error())
		panic(err)
	}

	for _, id := range ids {
		if id == 0 {
			continue
		}

		log.Printf("INFO: Attempting to download assets for class %d", id)
		os.MkdirAll(fmt.Sprintf("%s/training/%d", shims.SingleValueDiscardError(os.Getwd()), id), 0755)

		log.Printf("INFO: Getting narrative for class %d", id)
		err = a.DownloadTrainingNarrative(id, fmt.Sprintf("%s/training/%d/narrative.json", shims.SingleValueDiscardError(os.Getwd()), id))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
		}

		log.Printf("INFO: Getting attendance for class %d", id)
		err = a.DownloadTrainingAttendance(id, fmt.Sprintf("%s/training/%d/attendance.json", shims.SingleValueDiscardError(os.Getwd()), id))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
		}

		dest := fmt.Sprintf("%s/training/%d", shims.SingleValueDiscardError(os.Getwd()), id)
		os.MkdirAll(dest, 0755)

		err = a.DownloadTrainingAssets(id, dest)
		if err != nil {
			log.Printf("ERR: %s", err.Error())
		}

		//log.Printf("DEBUG: Wait 5 seconds")
		//time.Sleep(5 * time.Second)
	}
}
