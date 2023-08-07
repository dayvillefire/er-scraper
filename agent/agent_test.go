package agent

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var (
	oneTimeSetupRun = false
)

func testOneTimeSetup(t *testing.T) error {
	if oneTimeSetupRun {
		return nil
	}
	err := godotenv.Load()
	if err != nil {
		return err
	}
	oneTimeSetupRun = true
	return nil
}

func testGetAgent(t *testing.T) (*Agent, error) {
	err := testOneTimeSetup(t)
	if err != nil {
		return &Agent{}, err
	}

	a := &Agent{
		//Debug:    true,
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}
	err = a.Init()
	return a, err
}

func Test_Agent(t *testing.T) {
	_, err := testGetAgent(t)
	if err != nil {
		t.Fatalf("ERR: %s", err.Error())
	}
}
