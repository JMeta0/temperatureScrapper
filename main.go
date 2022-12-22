package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JMeta0/gothingspeak"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func getTemperature(host string) string {
	// Send an HTTP request to the URL and retrieve the response body
	resp, err := http.Get(host)
	if err != nil {
		log.Fatal("Cannot get temperature from host %s.\n%s", host, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Cannot read getTemperature response body.\n%s", err)
	}
	return string(body)
}

func sendViaSsh(keyPath string, host string, command string) string {
	// Read the private key file
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Println("Error while reading key file.\n%s", err)
	}

	// Create the Signer for the private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Println("Error while creating signer for the private key.\n%s", err)
	}

	// Set up the SSH config
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote host
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		log.Println("Error connecting to remote host.\n%s", err)
	}
	defer client.Close()

	// Open a new session
	session, err := client.NewSession()
	if err != nil {
		log.Println("Error opening console session.\n%s", err)
	}
	defer session.Close()

	// Output from command
	output, err := session.Output(command)
	if err != nil {
		log.Println("Error getting output from console.\n%s", err)
	}

	return string(output)
}

func thingspeak(temperature string, apiKey string) {
	ts := gothingspeak.NewChannelWriter(apiKey, true)
	if !ts.AddField(1, temperature) {
		return
	}
	_, err := ts.Update()
	if err != nil {
		log.Println("Couldn't update thingspeak.\n%s", err)
	}
}

func thingsboard(temperature string, domain string, apiKey string) {
	value := map[string]string{"temperature": temperature}
	json_data, err := json.Marshal(value)
	if err != nil {
		log.Println("Failed to marshal temperature data.\n%s", err)
	}
	thingsboardLink := fmt.Sprintf("https://%s/api/v1/%s/telemetry", domain, apiKey)
	_, err = http.Post(thingsboardLink, "application/json", bytes.NewBuffer(json_data))

	if err != nil {
		log.Println("Failed to send data to ThingsBoard.\n%s", err)
	}
}

func main() {
	////
	// Path of SSH private key to connect to host
	keyPath := "/home/user/.ssh/id_rsa"
	// Address of host
	host := "192.168.1.3:22"
	// Address of temperature sensor
	temperatureHost := "http://192.168.1.4"
	// ThingSpeak settings
	thingspeakApiKey := "API-KEY"
	// Thingsboard settings
	thingsboardApiKey := "API-KEY"
	thingsboardDomain := "thingsboard.example.com"
	////

	// Get the temperature
	temperature := getTemperature(temperatureHost)

	// Create command to execute on remote host
	command := fmt.Sprintf("echo \"%s - %s\" > /var/www/html/index.html", time.Now().Format(time.UnixDate), temperature)

	// Send temperature to Hetzner
	sendViaSsh(keyPath, host, command)

	// Send temperature to Thingspeak
	thingspeak(temperature, thingspeakApiKey)

	// send temperature to Thingsboard
	thingsboard(temperature, thingsboardDomain, thingsboardApiKey)
}
