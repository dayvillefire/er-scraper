package agent

import "testing"

/*
func Test_Agent_Refresh(t *testing.T) {
	a := Agent{
		Username: DEFAULT_USERNAME,
		Password: DEFAULT_PASSWORD,
		LoginUrl: DEFAULT_URL,
	}
	err := a.Init()
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

}
*/

func Test_Agent(t *testing.T) {
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

	c, err := a.authorizedGet("https://secure.emergencyreporting.com/training/classes.php")
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

	t.Logf("%s", c)

	err = a.DownloadTrainingAssets(7983393)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}

}
