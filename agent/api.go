package agent

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	classesOut, err := a.authorizedApiGetCall("https://secure.emergencyreporting.com/", csvurl)
	//classesOut, err := a.authorizedJsonGet2(csvurl)
	if err != nil {
		return out, fullout, err
	}

	return a.GetAllTrainingClassIDsFromCSV(classesOut)
}

func (a *Agent) GetAllTrainingClassIDsFromCSV(csvdata []byte) ([]int, [][]string, error) {
	out := make([]int, 0)
	fullout := [][]string{}

	reader := csv.NewReader(bytes.NewReader(csvdata))
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

	err = os.WriteFile(destFile, narrative, 0644)
	if err != nil {
		log.Printf("DownloadTrainingNarrative(): ERR: destfile = %s: %s", destFile, err.Error())
	}
	return err
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
		/*
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
		*/

		out, err := a.authorizedNativeGet(fmt.Sprintf(
			"https://secure.emergencyreporting.com/filedownload.php?fileguid=%s&contentdisposition=attachment",
			guid,
		))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}
		err = os.WriteFile(destPath+string(os.PathSeparator)+fn, out, 0644)
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}

		//log.Printf("DEBUG: Wait 2 seconds")
		//time.Sleep(2 * time.Second)
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
		pRaw, err := a.authorizedNativeGet(fmt.Sprintf("https://secure.emergencyreporting.com/nfirs/main_results.asp?pagenumber=%d", page))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			break
		}

		gq, err := goquery.NewDocumentFromReader(bytes.NewBuffer(pRaw))
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			break
		}

		gq.Find("button#button4").Each(func(i int, s *goquery.Selection) {
			_, exists := s.Attr("disabled")
			if exists {
				// Don't process beyond this page if the next button is disabled
				log.Printf("INFO: Found disabled next button on page %d", page)
				next = false
			}
		})

		pageIdMap := map[string]string{}

		gq.Find("td.listout").Each(func(i int, s *goquery.Selection) {
			onclick, exists := s.Attr("onclick")
			if !exists {
				//log.Printf("WARN: onclick doesn't exist in : %s", shims.SingleValueDiscardError(s.Html()))
				return
			}
			if strings.Index(onclick, "'") == -1 {
				//log.Printf("WARN: onclick is empty in : %s", shims.SingleValueDiscardError(s.Html()))
				return
			}
			pageIdMap[strings.Split(onclick, "'")[1]] = strings.Split(onclick, "'")[1]
		})

		page++

		if !next {
			log.Printf("INFO: No next page, breaking out of loop")
			break
		}

		allids = append(allids, shims.Values(pageIdMap)...)

		log.Printf("INFO: Collected %d ids", len(allids))
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

func (a *Agent) DownloadIncident(path string, eid string) error {
	// Determine if we're pulling a PDF file for patient information or not.
	// If not, just pull the HTML and grab images as well

	/*
		// TODO: This needs detection logic for patients
		{
			pdialog, err := a.authorizedNativeGet(fmt.Sprintf("https://secure.emergencyreporting.com/nfirs/print_form.asp?eid=%s&pid=&cid=&fromSummary=TRUE", eid))
			if err != nil {
				return err
			}

			gq, err := goquery.NewDocumentFromReader(bytes.NewBuffer(pdialog))
			if err != nil {
				return err
			}

		}
	*/

	os.MkdirAll(path, 0755) // silently ignore errors if already exists

	{
		phtml, err := a.authorizedNativeGet(fmt.Sprintf("https://secure.emergencyreporting.com/nfirs/print.asp?printtype=2&printtype=3&printtype=4&printtype=5&printtyperadio=5a&eid=%s&printtype=&printOption=&fromsummary=TRUE&cid=&patientcount=&notpayroll=TRUE", eid))
		if err != nil {
			return err
		}

		err = os.WriteFile(path+string(os.PathSeparator)+"incident.html", phtml, 0644)
		if err != nil {
			log.Printf("ERR: Was not able to write incident data: %s", err.Error())
			return nil
		}

		gq, err := goquery.NewDocumentFromReader(bytes.NewBuffer(phtml))
		if err != nil {
			return err
		}

		aCount := 0
		gq.Find("a").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}
			data, err := a.authorizedNativeGet("https://secure.emergencyreporting.com" + href)
			if err != nil {
				log.Printf("ERR: Was not able to fetch attachment %s: %s", href, err.Error())
				return
			}

			// TODO: IMPLEMENT: XXX: Extract the GUID and name the file that
			// TODO: IMPLEMENT: XXX: Type detection for files?

			err = os.WriteFile(path+string(os.PathSeparator)+fmt.Sprintf("attachment-%03d", aCount), data, 0644)
			if err != nil {
				log.Printf("ERR: Was not able to write attachment %s: %s", href, err.Error())
				return
			}
			aCount++
		})
	}

	// https://secure.emergencyreporting.com/nfirs/print.asp?printtype=2&printtype=3&printtype=4&printtype=5&printtyperadio=5a&eid=EID&printtype=&printOption=&fromsummary=TRUE&cid=&patientcount=&notpayroll=TRUE

	return nil
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
