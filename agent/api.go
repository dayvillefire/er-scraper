package agent

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

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
	url := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_people.php?classid=%d&_function=list_json", classId)
	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load class attendance list WS")
	attendance, err := a.authorizedJsonGet2(url)
	if err != nil {
		return err
	}

	return os.WriteFile(destFile, attendance, 0644)
}

func (a *Agent) DownloadTrainingNarrative(classId int, destFile string) error {
	url := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_narrative.php?classid=%d&_function=read", classId)
	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load class narrative WS")
	narrative, err := a.authorizedJsonGet2(url)
	if err != nil {
		return err
	}

	return os.WriteFile(destFile, narrative, 0644)
}

// DownloadTrainingAssets downloads training files, with appropriate names,
// to the specified destination path for the given class ID
func (a *Agent) DownloadTrainingAssets(classId int, destPath string) error {
	//url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class.php?id=%d&recurrence_mode=Single", classId)
	//url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class_files.php?id=%d&recurrence_mode=Single", classId)
	url := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_files.php?classid=%d&_function=list_json", classId)
	var err error

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Find files for class %d (url = %s)", classId, url)

	log.Printf("INFO: Load class file list WS")
	classfile, err := a.authorizedJsonGet2(url)
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
	url := "https://secure.emergencyreporting.com/webservices/admin/users.php?_function=list_json&_search=false&rows=500&page=1&sidx=name&sord=asc"

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load user list WS")
	users, err := a.authorizedJsonGet2(url)
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
