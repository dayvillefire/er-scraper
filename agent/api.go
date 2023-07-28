package agent

import (
	"fmt"
	"log"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
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

func (a *Agent) DownloadTrainingAssets(classId int) error {
	//url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class.php?id=%d&recurrence_mode=Single", classId)
	url := fmt.Sprintf("https://secure.emergencyreporting.com/training/class_files.php?id=%d&recurrence_mode=Single", classId)
	var nodes []*cdp.Node
	if err := chromedp.Run(a.ctx, chromedp.Navigate(url),
		chromedp.Tasks{
			chromedp.Nodes("div#file_img a", &nodes),
		}); err != nil {
		return fmt.Errorf("could not get url %s: %s", url, err.Error())
	}
	if len(nodes) < 1 {
		return fmt.Errorf("node not found")
	}
	href, found := nodes[0].Attribute("href")
	if !found {
		return fmt.Errorf("node not found")
	}

	out, err := a.authorizedDownload("https://secure.emergencyreporting.com" + href)
	if err != nil {
		return err
	}
	log.Printf("len(out) = %d", len(out))

	return nil // TODO: FIXME: XXX: IMPLEMENT
}
