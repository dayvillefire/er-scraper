package agent

import (
	"testing"
)

func testGetAgent(t *testing.T) (*Agent, error) {
	a := &Agent{
		//Debug:    true,
		Username: DEFAULT_USERNAME,
		Password: DEFAULT_PASSWORD,
		LoginUrl: DEFAULT_URL,
	}
	err := a.Init()
	return a, err
}

func Test_Agent(t *testing.T) {
	_, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
}
