package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/joho/godotenv"
)

type ipRes struct {
	Ip string `json:"ip"`
}

func GetIp() (string, error) {
	response, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		log.Fatalln("GetIp", err)
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Status)

	jsonResponse := ipRes{}
	unmarshaLerr := json.Unmarshal(responseBody, &jsonResponse)
	if unmarshaLerr != nil {
		log.Fatal(err)
	}

	fmt.Println("external ip", jsonResponse.Ip)

	return jsonResponse.Ip, nil
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	domain := os.Getenv("CLOUDFLARE_DOMAIN")

	ip, err := GetIp()
	if err != nil {
		log.Fatal("GetIp", err)
	}

	api, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	if err != nil {
		log.Fatal("NewWithAPIToken", err)
	}

	// Most API calls require a Context
	ctx := context.Background()

	id, err := api.ZoneIDByName(domain) // example.com
	if err != nil {
		log.Fatal("ZoneIDByName", err)
	}

	fmt.Println("zone id:", id)

	zone, err := api.ZoneDetails(ctx, id)
	if err != nil {
		log.Fatal("ZoneDetails", err)
	}

	subDomains := strings.Split(os.Getenv("CLOUDFLARE_SUB_DOMAINS"), ",")

	fmt.Println("subDomains: ", subDomains)
	fmt.Println("domain", zone.Name)

	for _, subDomain := range subDomains {
		var host string

		if subDomain == "@" {
			host = zone.Name
		} else {
			host = subDomain + "." + zone.Name
		}

		fmt.Println("Zone Value for", subDomain, "is", host)

		recordFilter := cloudflare.DNSRecord{
			Type: "A",
			Name: host,
		}

		records, err := api.DNSRecords(ctx, id, recordFilter)
		if err != nil {
			log.Fatal("DNSRecords", err)
		}
		fmt.Println("records", records)

		var record cloudflare.DNSRecord

		if len(records) == 0 {
			fmt.Println("Record not found, creating...")
			newRecord := cloudflare.DNSRecord{
				Type:    "A",
				Name:    host,
				Content: ip,
			}
			_, err := api.CreateDNSRecord(ctx, id, newRecord)
			if err != nil {
				log.Fatal("CreateDNSRecord", err)
			}

			fmt.Println("Succsfully created record for host", host, "with ip", ip)
			continue
		}

		if len(records) > 1 {
			fmt.Println("Records", records, "for", host, "longer then 1, continuing...")
			continue
		} else {
			record = records[0]
		}

		fmt.Println("Got host record with ID", record.ID, "for host", record.Name, ", current value", record.Content)

		if record.Content != ip {
			newRecord := cloudflare.DNSRecord{
				Content: ip,
			}

			err = api.UpdateDNSRecord(ctx, id, record.ID, newRecord)
			if err != nil {
				fmt.Println("Error updating IP for host\nError:", err)
				continue
			} else {
				fmt.Println("Succsfully updated record for host", host, "with ip", ip)
			}
		}
	}
}
