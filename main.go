package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var conf *config

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting IFTTT relay..")

	if err := loadConfiguration(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	http.HandleFunc("/minecraft", handleMinecraft)
	log.Fatal(http.ListenAndServe(conf.ServerAddress, nil))
}

type config struct {
	WebhookKey    string
	ServerAddress string
}

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

func handleMinecraft(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()
	log.Println("Recieved minecraft request")

	resp, err := http.Get("http://minecraft.netsoc.co/standalone/dynmap_NetsocCraft.json")
	if err != nil {
		err := fmt.Sprintf("Failed to retrieve data from the Minecraft Server: %v", err)
		log.Println(err)
		http.Error(w, err, 500)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Sprintf("Failed to read the minecraft response body: %v", err)
		log.Println(err)
		http.Error(w, err, 500)
		return
	}

	q := &struct {
		Players []struct {
			Name string `json:"name"`
		} `json:"players"`
		Updates []interface{} `json:"updates"`
	}{}

	if err := json.Unmarshal(body, q); err != nil {
		err := fmt.Sprintf("Failed to parse response json %q: %v", string(body), err)
		log.Println(err)
		http.Error(w, err, 500)
		return
	}

	numPlayers := len(q.Players)
	url := fmt.Sprintf("https://maker.ifttt.com/trigger/minecraft/with/key/%s?value1=%d",
		conf.WebhookKey, numPlayers)
	resp, err = http.Get(url)
	if err != nil {
		err := fmt.Sprintf("Failed to make webhook trigger: %v", err)
		log.Println(err)
		http.Error(w, err, 500)
		return
	}
	defer resp.Body.Close()
	log.Printf("Responded with %d people", numPlayers)
}
