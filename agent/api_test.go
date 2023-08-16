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

func Test_GetUsers(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	users, err := a.GetUsers()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	records, found := users["records"]
	if !found {
		t.Fatalf("ERR: Returned %#v", users)
	}

	t.Logf("INFO: Found %s user records", records)
}

// GetUserCertifications
func Test_GetUserCertifications(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	data, err := a.GetUserCertifications(411472)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	records, found := data["totalRows"]
	if !found {
		t.Fatalf("ERR: Returned %#v", data)
	}

	t.Logf("INFO: Found %s user records", records)
}

func Test_GetHydrants(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	data, err := a.GetHydrants()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	t.Logf("INFO: Found %d hydrant records", len(data))
}

func Test_GetIncidentIDs(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	data, err := a.GetIncidentIDs()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	t.Logf("INFO: Found %d incident records", len(data))
}

func Test_ExportCalendar(t *testing.T) {
	a, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	data, err := a.ExportCalendar()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	t.Logf("INFO: %s", string(data))
}
