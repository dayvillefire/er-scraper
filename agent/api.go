package agent

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/chromedp/chromedp"
	"github.com/jbuchbinder/shims"
)

func (a *Agent) IsAuthorized() error {
	return nil
}

// GetAllTrainingClassIDs returns a list of all training class records in the system
func (a *Agent) GetAllTrainingClassIDs() ([]int, [][]string, error) {
	out := make([]int, 0)
	fullout := [][]string{}

	a.ContextSwitch = ContextDownload

	csvurl := "https://secure.emergencyreporting.com/training/ws/classes.php?_function=list_csv&_csvtype=info"

	log.Printf("INFO: Load class list WS")
	classesOut, err := a.authorizedDownload(csvurl)
	if err != nil {
		return out, fullout, err
	}

	log.Printf("INFO: CSV temporary file: %s", classesOut)
	defer os.Remove(classesOut)

	classesFp, err := os.Open(classesOut)
	if err != nil {
		return out, fullout, err
	}

	reader := csv.NewReader(classesFp)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, fullout, err
		}
		fullout = append(fullout, record)
		out = append(out, shims.SingleValueDiscardError(strconv.Atoi(record[0])))
	}

	return out, fullout, nil
}

func (a *Agent) DownloadTrainingAttendance(classId int, destFile string) error {
	u := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_people.php?classid=%d&_function=list_json", classId)
	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load class attendance list WS")
	attendance, err := a.authorizedJsonGet2(u)
	if err != nil {
		return err
	}

	return os.WriteFile(destFile, attendance, 0644)
}

func (a *Agent) DownloadTrainingNarrative(classId int, destFile string) error {
	u := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_narrative.php?classid=%d&_function=read", classId)
	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load class narrative WS")
	narrative, err := a.authorizedJsonGet2(u)
	if err != nil {
		return err
	}

	return os.WriteFile(destFile, narrative, 0644)
}

// DownloadTrainingAssets downloads training files, with appropriate names,
// to the specified destination path for the given class ID
func (a *Agent) DownloadTrainingAssets(classId int, destPath string) error {
	//u := fmt.Sprintf("https://secure.emergencyreporting.com/training/class.php?id=%d&recurrence_mode=Single", classId)
	//u := fmt.Sprintf("https://secure.emergencyreporting.com/training/class_files.php?id=%d&recurrence_mode=Single", classId)
	u := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_files.php?classid=%d&_function=list_json", classId)
	var err error

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Find files for class %d (url = %s)", classId, u)

	log.Printf("INFO: Load class file list WS")
	classfile, err := a.authorizedJsonGet2(u)
	if err != nil {
		return err
	}

	fileMap := map[string]string{}
	fileGuidMap := map[string]string{}

	{
		type classResponse struct {
			Rows []struct {
				Id   string   `json:"id"`
				Cell []string `json:"cell"`
			} `json:"rows"`
		}

		var cr classResponse
		err = json.Unmarshal(classfile, &cr)
		if err != nil {
			return err
		}

		for _, r := range cr.Rows {
			if len(r.Cell) < 3 {
				continue
			}
			fileMap[r.Cell[0]] = r.Id
		}

		log.Printf("filemap = %#v", fileMap)
	}

	for fn, id := range fileMap {
		classFileInfo, err := a.authorizedJsonGet2(
			fmt.Sprintf(
				"https://secure.emergencyreporting.com/training/ws/class_files.php?classid=%d&id=%s&_function=detail",
				classId, id,
			))

		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}

		/*
			cfi := string(classFileInfo)
			cfi = strings.TrimPrefix(cfi, "<head></head><body>")
			cfi = strings.TrimSuffix(cfi, "</body>")
			cfi = strings.TrimSuffix(cfi, "</span>")
		*/

		type classFileInfoType struct {
			Accesslevel string `json:"accesslevel"`
			Description string `json:"description"`
			Fileguid    string `json:"fileguid"`
			Name        string `json:"name"`
			Url         string `json:"url"`
		}

		if a.Debug {
			log.Printf("DEBUG: CFI = %s", string(classFileInfo))
		}

		var cfiOut classFileInfoType
		err = json.Unmarshal(classFileInfo, &cfiOut)
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}

		if a.Debug {
			log.Printf("DEBUG: CFI = %v, fn = %s", cfiOut, fn)
		}

		fileGuidMap[fn] = cfiOut.Fileguid
	}

	log.Printf("INFO: fileguidmap = %#v", fileGuidMap)

	for fn, guid := range fileGuidMap {
		var out string
		out, err = a.authorizedDownload(fmt.Sprintf(
			"https://secure.emergencyreporting.com/filedownload.php?fileguid=%s&contentdisposition=attachment",
			guid,
		))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}
		log.Printf("INFO: title = %s, temp file = %s", fn, out)
		err = os.Rename(out, destPath+string(os.PathSeparator)+fn)
		if err != nil {
			log.Printf("ERR: renaming file %s to %s: %s", out, destPath+string(os.PathSeparator)+fn, err.Error())
		}
	}

	return err
}

func (a *Agent) GetUsers() (map[string]any, error) {
	out := make(map[string]any, 0)
	u := "https://secure.emergencyreporting.com/webservices/admin/users.php?_function=list_json&_search=false&rows=500&page=1&sidx=name&sord=asc"

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load user list WS")
	users, err := a.authorizedJsonGet2(u)
	if err != nil {
		log.Printf("ERR: %s: %s", err.Error(), string(users))
		return out, err
	}

	err = json.Unmarshal(users, &out)
	if err != nil {
		log.Printf("ERR: %s: %s", err.Error(), string(users))
	}
	return out, err
}

func (a *Agent) GetUserCertifications(userId int) (map[string]any, error) {
	out := make(map[string]any, 0)
	u := fmt.Sprintf("https://api.emergencyreporting.com/V1/users/%d/certifications?limit=1000", userId)

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load user certifications WS")
	data, err := a.authorizedApiGetCall(
		fmt.Sprintf("https://secure.emergencyreporting.com/admin_user/users/Certifications.php?userid=%d", userId),
		u,
	)

	if err != nil {
		log.Printf("ERR: %s: %s", err.Error(), string(data))
		return out, err
	}

	err = json.Unmarshal(data, &out)
	if err != nil {
		log.Printf("ERR: %s: %s", err.Error(), string(data))
	}
	return out, err
}

// GetHydrants returns an array of all hydrant data
func (a *Agent) GetHydrants() ([][]string, error) {
	return a.getCsvUrl("https://secure.emergencyreporting.com/webservices/hydrants/hydrants.php?_type=hydrants&_function=list_csv")
}

// GetIncidentIDs returns an array of all incident data
func (a *Agent) GetIncidentIDs() ([]string, error) {
	var target any // temporary holding spot -- we just discard this
	if err := chromedp.Run(a.ctx,
		chromedp.Navigate("https://secure.emergencyreporting.com/nfirs/main.asp"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Waiting for header frame to be visible")
			return nil
		}),
		chromedp.WaitVisible("//frameset/frame[@src='searchoptions.asp']"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Setting multioption")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('input[id="Radio1"]').checked = true;`, &target),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Setting search range")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('select[name="searchDateRange"]').value = 'AllTime';`, &target),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Submitting form")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('input[id="Submit2"]').click();`, &target),
	); err != nil {
		return []string{}, err
	}

	allids := make([]string, 0)

	next := true
	page := 1
	// Enter loop
	for {
		var disabled bool
		if err := chromedp.Run(a.ctx,
			chromedp.Evaluate(`top.frames[2].document.querySelector('button#button3').disabled;`, &disabled),
		); err != nil {
			log.Printf("ERR: %s", err.Error())
			break
		}
		// Don't process beyond this page if the next button is disabled
		next = !disabled

		// Get the list of IDs
		var ids []string
		if err := chromedp.Run(a.ctx,
			chromedp.Evaluate(`Array.prototype.slice.call(top.frames[2].document.querySelectorAll('td.listout:nth-child(1)')).map(val => val.getAttribute("onclick").split("'")[1]);`, &ids),
		); err != nil {
			log.Printf("ERR: %s", err.Error())
			break
		}
		allids = append(allids, ids...)
		log.Printf("INFO: Found %d ids on page %d, total %d", len(ids), page, len(allids))

		page++

		if !next {
			log.Printf("INFO: No next page, breaking out of loop")
			break
		}

		var target any
		if err := chromedp.Run(a.ctx,
			chromedp.Evaluate(fmt.Sprintf("window.frames[2].location='?pagenumber=%d';", page), &target), // .getElementById('button3').click();`, &target),
		); err != nil {
			log.Printf("ERR: Submitting next page: %s", err.Error())
			break
		}
		//log.Printf("DEBUG: Target = %#v", target)
		//break

	}

	return allids, nil
}

// GetIncidentsCsv returns an array of all incident data
func (a *Agent) GetIncidentsCSV() ([][]string, error) {
	var target any // temporary holding spot -- we just discard this
	if err := chromedp.Run(a.ctx,
		chromedp.Navigate("https://secure.emergencyreporting.com/nfirs/main.asp"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Waiting for header frame to be visible")
			return nil
		}),
		chromedp.WaitVisible("//frameset/frame[@src='searchoptions.asp']"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Setting multioption")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('input[id="Radio1"]').checked = true;`, &target),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Setting search range")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('select[name="searchDateRange"]').value = 'AllTime';`, &target),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("INFO: Submitting form")
			return nil
		}),
		chromedp.Evaluate(`top.frames[1].document.querySelector('input[id="Submit2"]').click();`, &target),
	); err != nil {
		return [][]string{}, err
	}

	return a.getCsvUrl("https://secure.emergencyreporting.com/nfirs/main_results.asp?downloadCSV=1")
}

// exposureform
// POST https://secure.emergencyreporting.com/nfirs/includes/top_main.asp?csrt=1772698412475071612
// hidden iid
// hidden eredirectto
// hidden eid

func (a *Agent) ExportCalendar() ([]byte, error) {
	v := url.Values{}
	v.Set("exportType", "ics")
	v.Set("StartDate", "01/01/2005")
	v.Set("EndDate", "01/01/2025")
	v.Set("EntryTypes", "")

	u := "https://secure.emergencyreporting.com/calendar/includes/backends/calendar_export.php?" + v.Encode()

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load calendar WS")
	oFile, err := a.authorizedDownload(u)

	if err != nil {
		log.Printf("ERR: %s: %s", err.Error(), oFile)
		return []byte{}, err
	}

	log.Printf("INFO: temporary file: %s", oFile)
	defer os.Remove(oFile)

	return os.ReadFile(oFile)
}
