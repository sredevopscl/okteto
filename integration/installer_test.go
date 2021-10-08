package integration

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

var (
	name         = ""
	installerUrl = ""
	terraformEx  = "terraform"
)

func TestMain(t *testing.T) {
	checkEnv()
	if _, err := exec.LookPath(terraformEx); err != nil {
		log.Fatalf("terraform is not in the path: %s", err)
		os.Exit(1)
	}

	// TERRAFORM INIT IN WORKDIR
	args := []string{"init", installerUrl}
	tfCommand(args)

	// TERRAFORM APPLY IN WORKDIR WITH TFVARS AND CONFIG.YAML NEEDED
	args = []string{"apply", "-auto-approve"}
	tfCommand(args)

	// HIT ENDPOINT
	upErrorChannel := make(chan error, 1)
	endpoint := fmt.Sprintf("https://okteto.%s.dev.okteto.net/healthz", name)

	_, err := getContent(endpoint, 1200, upErrorChannel)
	if err != nil {
		t.Fatalf("failed to get index content: %s", err)
	} else {
		log.Println("Success finding endpoint")
	}
}

func tfCommand(myarg []string) {
	cmd := exec.Command("terraform", myarg...)
	cmd.Dir = installerUrl
	outInit, err := cmd.Output()
	log.Printf("launching command: %s", cmd.String())
	printOrErr(outInit, err)
}

func checkEnv() {
	if nameokt, ok := os.LookupEnv("NAME"); !ok {
		log.Println("NAME is not defined")
		os.Exit(1)
	} else {
		name = nameokt
	}

	if urlInst, ok := os.LookupEnv("URL_INSTALLER"); !ok {
		log.Println("URL_INSTALLER is not defined")
		os.Exit(1)
	} else {
		installerUrl = urlInst
	}

	if _, ok := os.LookupEnv("TF_VAR_credentials"); !ok {
		log.Println("TF_VAR_credentials is not defined")
		os.Exit(1)
	}
}

func printOrErr(out []byte, err error) {
	fmt.Printf("%s", out)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func getContent(endpoint string, timeout int, upErrorChannel chan error) (string, error) {
	t := time.NewTicker(1 * time.Second)
	for i := 0; i < timeout; i++ {
		r, err := http.Get(endpoint)
		if err != nil {
			log.Printf("called %s, got %s, retrying", endpoint, err)
			<-t.C
			continue
		}

		defer r.Body.Close()
		if r.StatusCode != 200 {
			log.Printf("called %s, got status %d, retrying", endpoint, r.StatusCode)
			<-t.C
			continue
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}

		return string(body), nil
	}

	return "", fmt.Errorf("service wasn't available")
}
