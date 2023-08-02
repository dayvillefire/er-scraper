package agent

import "testing"

func Test_GetAllTrainingClassIDs(t *testing.T) {
	a := Agent{
		//Debug:    true,
		Username: DEFAULT_USERNAME,
		Password: DEFAULT_PASSWORD,
		LoginUrl: DEFAULT_URL,
	}
	err := a.Init()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	ids, err := a.GetAllTrainingClassIDs()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
	t.Logf("ids = %#v", ids)
}
