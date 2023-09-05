package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/libdns/libdns"
	"github.com/libdns/porkbun"
)

func main() {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error loading .env file", envErr)
	}

	apikey := os.Getenv("PORKBUN_API_KEY")
	secretapikey := os.Getenv("PORKBUN_SECRET_API_KEY")
	zone := os.Getenv("ZONE")

	if apikey == "" || secretapikey == "" || zone == "" {
		fmt.Println("All variables must be set in '.env' file")
		return
	}

	provider := porkbun.Provider{
		APIKey:       apikey,
		APISecretKey: secretapikey,
	}

	//Check Authorization
	_, err := provider.CheckCredentials(context.TODO())

	if err != nil {
		log.Fatalf("Credential check failed: %s\n", err.Error())
	}

	//Get records
	initialRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalf("Failed to fetch records: %s\n", err.Error())
	}

	log.Println("Records fetched:")
	for _, record := range initialRecords {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
	}

	testValue := "test-value"
	updatedTestValue := "updated-test-value"
	ttl := time.Duration(600 * time.Second)
	recordType := "TXT"
	testFullName := "libdns_test_record." + zone

	//Create record
	appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
		{
			Type:  recordType,
			Name:  testFullName,
			TTL:   ttl,
			Value: testValue,
		},
	})

	if err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}

	//Get records
	postCreatedRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalf("Failed to fetch records: %s\n", err.Error())
	}

	if len(postCreatedRecords) != len(initialRecords)+1 {
		log.Fatalln("Additional record not created")
	}

	fmt.Printf("Created record: \n%v\n", appendedRecords[0])

	// Update record
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{
		{
			Type:  recordType,
			Name:  testFullName,
			TTL:   ttl,
			Value: updatedTestValue,
		},
	})

	if err != nil {
		log.Fatalf("ERROR: %s\n", err.Error())
	}
	fmt.Printf("Updated record: \n%v\n", updatedRecords[0])

	//Get records
	updatedRecords, err = provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalf("Failed to fetch records: %s\n", err.Error())
	}

	if len(updatedRecords) != len(initialRecords)+1 {
		log.Fatalln("Additional record created instead of updating existing")
	}

	// Delete record
	deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{
		{
			Type: recordType,
			Name: testFullName,
		},
	})

	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}

	//Get records
	updatedRecords, err = provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalf("Failed to fetch records: %s\n", err.Error())
	}

	if len(updatedRecords) != len(initialRecords) {
		log.Fatalln("Additional record not cleaned up")
	}

	fmt.Printf("Deleted record: \n%v\n", deleteRecords[0])

}
