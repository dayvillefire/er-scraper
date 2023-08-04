package agent

import (
	"os"
	"testing"

	"github.com/jbuchbinder/shims"
)

func Test_GetAllTrainingClassIDs(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	ids, _, err := a.GetAllTrainingClassIDs()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
	t.Logf("ids = %#v", ids)
}

func Test_DownloadTrainingAssets(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	err = a.DownloadTrainingAssets(7988356, shims.SingleValueDiscardError(os.Getwd())) // 7983393)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
}

func Test_DownloadTrainingNarrative(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	err = a.DownloadTrainingNarrative(5897010, shims.SingleValueDiscardError(os.Getwd())+"/narrative.txt")
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
}

func Test_DownloadTrainingAttendance(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	err = a.DownloadTrainingAttendance(5897010, shims.SingleValueDiscardError(os.Getwd())+"/attendance.txt")
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
}
