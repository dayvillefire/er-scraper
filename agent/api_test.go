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

	ids, err := a.GetAllTrainingClassIDs()
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
