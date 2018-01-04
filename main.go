package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var conf *config

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting IFTTT relay..")

	if err := loadConfiguration(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	http.HandleFunc("/minecraft", handleMinecraft)
	http.HandleFunc("/webtext", handleWebtext)
	log.Fatal(http.ListenAndServe(conf.ServerAddress, nil))
}

type config struct {
	WebhookKey    string
	ServerAddress string
	WebtextBinary string
}

// loadConfiguration loads the JSON configuration file
func loadConfiguration() error {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("failed to read json configuration file: %v", err)
	}

	conf = &config{}
	if err := json.Unmarshal(data, conf); err != nil {
		return fmt.Errorf("failed to unmarshal config JSON: %v", err)
	}
	return nil
}

func handleWebtext(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Println("Received webtext request")
	raw, ok := r.URL.Query()["raw"]
	if !ok {
		errCode(w, "Raw not found in URL paramters", 400)
		return
	}

	if len(raw) != 1 {
		errCode(w, "Too many values for the 'raw' key", 400)
	}

	tmp := strings.Split(raw[0], " saying ")
	if len(tmp) != 2 {
		errCode(w, fmt.Sprintf("Raw command %q not formatted correctly", raw), 400)
		return
	}

	var (
		rec = tmp[0]
		msg = tmp[1]
	)
	cmd := exec.Command(conf.WebtextBinary, "-r", rec, msg)
	if out, err := cmd.CombinedOutput(); err != nil {
		errCode(w, fmt.Sprintf(
			"failed to run webtext command: %s", strings.Trim(string(out), "\n")), 500)
		return
	}

	if err := sendToPhone(fmt.Sprintf("Successfully sent %q to %q", msg, rec)); err != nil {
		errCode(w, fmt.Sprintf("failed to send confirmation: %v", err), 500)
		return
	}
	log.Printf("Sent %q to %q", msg, rec)
}

func handleMinecraft(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()
	log.Println("Recieved minecraft request")

	resp, err := http.Get("http://minecraft.netsoc.co/standalone/dynmap_NetsocCraft.json")
	if err != nil {
		errCode(w,
			fmt.Sprintf("Failed to retrieve data from the Minecraft Server: %v", err), 500)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errCode(w, fmt.Sprintf("Failed to read the minecraft response body: %v", err), 500)
		return
	}

	q := &struct {
		Players []struct {
			Name string `json:"name"`
		} `json:"players"`
		Updates []interface{} `json:"updates"`
	}{}

	if err := json.Unmarshal(body, q); err != nil {
		errCode(w,
			fmt.Sprintf("Failed to parse response json %q: %v", string(body), err), 500)
		return
	}

	var (
		numPlayers = len(q.Players)
		msg        string
	)
	switch numPlayers {
	case 0:
		msg = "No one is playing minecraft"
	case 1:
		msg = "One person is playing minecraft"
	default:
		msg = fmt.Sprintf("%d people playing minecraft", numPlayers)
	}

	if err := sendToPhone(msg); err != nil {
		errCode(w, fmt.Sprintf("failed to send notification: %v", err), 500)
	}

	log.Printf("Responded with %q", msg)
}

// errCode logs the error message and writes the error to the response
func errCode(w http.ResponseWriter, msg string, code int) {
	log.Println(msg)
	http.Error(w, msg, code)
}

// sendToPhone sends a notification to my phone through a webhook
func sendToPhone(msg string) error {
	url := fmt.Sprintf("https://maker.ifttt.com/trigger/notification/with/key/%s?value1=%s",
		conf.WebhookKey, msg)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to make webhook trigger: %v", err)
	}
	resp.Body.Close()
	return nil
}
