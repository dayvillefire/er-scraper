package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func (a *Agent) IsAuthorized() error {
	return nil
}

/*
func (a *Agent) GetAllTrainingClassIDs() ([]int, error) {
	csvurl := "https://secure.emergencyreporting.com/training/ws/classes.php?_function=list_csv&_csvtype=info"

	return nil // TODO: FIXME: XXX: IMPLEMENT
}
*/

func (a *Agent) DownloadTrainingAssets(destPath string, classId int) error {
	//url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class.php?id=%d&recurrence_mode=Single", classId)
	//url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class_files.php?id=%d&recurrence_mode=Single", classId)
	url := fmt.Sprintf("https://secure.emergencyreporting.com/training/ws/class_files.php?classid=%d&_function=list_json", classId)
	//var outernodes []*cdp.Node
	var err error
	//var nodes []*cdp.Node

	a.ContextSwitch = ContextDownload

	log.Printf("Find files for class %d (url = %s)", classId, url)

	log.Printf("Load class file list WS")
	classfile, err := a.authorizedGet(url)
	if err != nil {
		return err
	}

	fileMap := map[string]string{}
	fileGuidMap := map[string]string{}

	{
		// Fugly part 1 -- this is getting wrapped as HTML, even though it clearly isn't.
		cfs := string(classfile)
		cfs = strings.TrimPrefix(cfs, "<head></head><body>")
		cfs = strings.TrimSuffix(cfs, "</body>")
		cfs = strings.TrimSuffix(cfs, "</span>")
		cfs = strings.TrimSuffix(cfs, "</span>")
		cfs = strings.TrimSuffix(cfs, "</span>")
		cfs = strings.ReplaceAll(cfs, `""\&quot;`, "")
		cfs = strings.ReplaceAll(cfs, `<img src="\&quot;\/graphics\/trash2.gif\&quot;" alt="\&quot;Delete" uploaded="" file\"="" title="\&quot;Delete" class="\&quot;jqg_delete\&quot;">&lt;\/span&gt;`, "")
		cfs = strings.ReplaceAll(cfs, `"\&quot;`, "")
		cfs = strings.ReplaceAll(cfs, `\&quot;"`, "")

		// Fugly part 2 -- replace to pull out bad serialization
		//m1 := regexp.MustCompile(`<span class=".*\/span&gt;`)
		//cfs = m1.ReplaceAllString(cfs, "")

		log.Printf("FILE = %s", cfs)

		classfile = []byte(cfs)

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
		classFileInfo, err := a.authorizedGet(
			fmt.Sprintf(
				"https://secure.emergencyreporting.com/training/ws/class_files.php?classid=%d&id=%s&_function=detail",
				classId, id,
			))
		/*
			classFileInfo, err := a.authorizedPost("https://secure.emergencyreporting.com/training/ws/class_files.php", map[string]any{
				"classid":   classId,
				"id":        id,
				"_function": "detail",
			})
		*/
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}
		cfi := string(classFileInfo)
		cfi = strings.TrimPrefix(cfi, "<head></head><body>")
		cfi = strings.TrimSuffix(cfi, "</body>")
		cfi = strings.TrimSuffix(cfi, "</span>")

		type classFileInfoType struct {
			Accesslevel string `json:"accesslevel"`
			Description string `json:"description"`
			Fileguid    string `json:"fileguid"`
			Name        string `json:"name"`
			Url         string `json:"url"`
		}

		log.Printf("CFI = %s", cfi)

		var cfiOut classFileInfoType
		err = json.Unmarshal([]byte(cfi), &cfiOut)
		if err != nil {
			log.Printf("ERR: %s", err.Error())
			continue
		}

		log.Printf("CFI = %v, fn = %s", cfiOut, fn)

		fileGuidMap[fn] = cfiOut.Fileguid
	}

	log.Printf("fileguidmap = %#v", fileGuidMap)

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
		log.Printf("title = %s, temp file = %s", fn, out)
	}

	return err
}
