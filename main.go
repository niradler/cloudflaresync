package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func GetIp() (string, error) {
	response, err := http.Get("https://1.1.1.1/cdn-cgi/trace")
	if err != nil {
		log.Fatalln("GetIp", err)
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response.Status)
	resBody := string(responseBody)
	split := strings.Split(resBody, "\n")[2]
	ip := strings.Split(split, "=")[1]
	fmt.Println("external ip", ip)

	return ip, nil
}

func getContainersName() []string {
	var subs []string
	var useDockerDaemon = false
	useDockerDaemonEnv, exist := os.LookupEnv("DOCKER_DAEMON")
	if !exist || useDockerDaemonEnv == "false" {
		useDockerDaemon = false
	} else {
		useDockerDaemon = true
	}

	if useDockerDaemon == false {
		return nil
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		name, ok := container.Labels["cloudflaresync.name"]

		if ok {
			subs = append(subs, name)
		}
	}

	return subs
}

func updateRecords() {

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
	names := getContainersName()
	if names != nil {
		subDomains = append(subDomains, names...)
	}

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
			var proxied bool
			proxiedEnvVar, exist := os.LookupEnv("PROXIED")
			if !exist {
				proxied = true
			} else {
				boolValue, err := strconv.ParseBool(proxiedEnvVar)
				if err != nil {
					log.Fatal(err)
				}
				proxied = boolValue
			}
			newRecord := cloudflare.DNSRecord{
				Type:    "A",
				Name:    host,
				Content: ip,
				Proxied: &proxied,
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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	updateRecords()
	log.Println("Start cron", time.Now())
	cronExpression, exist := os.LookupEnv("CRON")
	if !exist {
		cronExpression = "0 * * * *"
	}
	log.Println("cron:", cronExpression)
	c := cron.New()
	id, err := c.AddFunc(cronExpression, updateRecords)
	if err != nil {
		log.Fatal("cron AddFunc error:", err)
	}
	log.Println("cron id:", id)
	c.Start()
	log.Println("Cron Info: ", c.Entries())

	go forever()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	fmt.Println("done")

}

func forever() {
	for {
		time.Sleep(time.Second)
	}
}
