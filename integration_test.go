package porkbun

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/libdns/libdns"
)

var records []libdns.Record
var testRecord *libdns.Record

func getInitialRecords(t *testing.T, provider Provider, zone string) []libdns.Record {
	if len(records) == 0 {
		fetchedRecords, err := provider.GetRecords(context.TODO(), zone)
		if err != nil {
			t.Error(err)
		}
		records = fetchedRecords
	}

	return records
}

func createOrGetTestRecord(t *testing.T, provider Provider, zone string) (*libdns.Record, error) {
	if testRecord == nil {
		testValue := "test-value"
		ttl := time.Duration(600 * time.Second)
		testFullName := "libdns_test_record"

		//Create record
		appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
			libdns.TXT{
				Name: testFullName,
				TTL:  ttl,
				Text: testValue,
			},
		})

		if err != nil {
			t.Error(err)
			t.Fail()
			return nil, err
		}

		if len(appendedRecords) != 1 {
			err = fmt.Errorf("Incorrect amount of records %d created", len(appendedRecords))
			t.Error(err)
			return nil, err
		} else {
			testRecord = &appendedRecords[0]
		}
	}

	return testRecord, nil
}

func createOrGetRootRecord(t *testing.T, provider Provider, zone string) libdns.Record {
	testValue := "test-value"
	ttl := time.Duration(600 * time.Second)
	testFullName := "@"

	//Create record
	appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
		libdns.CNAME{
			Name:   testFullName,
			TTL:    ttl,
			Target: testValue,
		},
	})

	if err != nil {
		t.Error(err)
		t.Fail()
	}

	if len(appendedRecords) != 1 {
		t.Errorf("Incorrect amount of records %d created", len(appendedRecords))
	}

	return appendedRecords[0]
}

func getProvider(t *testing.T) (Provider, string) {
	envErr := godotenv.Load()
	if envErr != nil {
		t.Error(envErr)
	}

	apikey := os.Getenv("PORKBUN_API_KEY")
	secretapikey := os.Getenv("PORKBUN_SECRET_API_KEY")
	zone := os.Getenv("ZONE")

	if apikey == "" || secretapikey == "" || zone == "" {
		t.Errorf("All variables must be set in '.env' file")
	}

	provider := Provider{
		APIKey:       apikey,
		APISecretKey: secretapikey,
	}
	return provider, zone
}

func TestProvider_CheckCredentials(t *testing.T) {
	provider, _ := getProvider(t)

	//Check Authorization
	_, err := provider.CheckCredentials(context.TODO())

	if err != nil {
		t.Error(err)
		t.Fatal()
	}
}

func TestProvider_GetRecords(t *testing.T) {
	provider, zone := getProvider(t)

	//Get records
	initialRecords := getInitialRecords(t, provider, zone)

	log.Println("Records fetched:")
	for _, record := range initialRecords {
		rr := record.RR()
		t.Logf("%s (.%s): %s, %s\n", rr.Name, zone, rr.Data, rr.Type)
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	provider, zone := getProvider(t)

	//Get records
	initialRecords := getInitialRecords(t, provider, zone)

	createdRecord, err := createOrGetTestRecord(t, provider, zone)
	if err != nil {
		t.Error(err)
		return
	}
	//Get records
	postCreatedRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Error(err)
	}

	if len(postCreatedRecords) != len(initialRecords)+1 {
		t.Errorf("Additional record not created. got: %d, wanted: %d\n", len(postCreatedRecords), len(initialRecords)+1)
	}

	t.Logf("Created record: \n%v\n", createdRecord)
}

func TestProvider_ModifyRootRecord(t *testing.T) {
	provider, zone := getProvider(t)

	//Get records
	initialRecords := getInitialRecords(t, provider, zone)

	createdRecord := createOrGetRootRecord(t, provider, zone)
	//Get records
	postCreatedRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Error(err)
	}

	if len(postCreatedRecords) != len(initialRecords)+1 {
		t.Errorf("Additional record not created")
	}

	t.Logf("Created record: \n%v\n", createdRecord)

	updatedTestValue := "updated-test-value"
	// Update record
	records := []libdns.Record{
		libdns.CNAME{
			Name:   "@",
			TTL:    time.Duration(600 * time.Second),
			Target: updatedTestValue,
		},
	}
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, records)

	if err != nil {
		t.Error(err)
	}

	if len(updatedRecords) != 1 {
		t.Logf("Incorrect amount of records changed")
	}

	t.Logf("Updated root record: \n%v\n", updatedRecords[0])

	deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{createdRecord})
	if err != nil {
		t.Error(err)
	}

	if len(deleteRecords) != 1 {
		t.Errorf("Deleted incorrect amount of records %d", len(deleteRecords))
	}

	t.Logf("Deleted record: \n%v\n", deleteRecords[0])
}

func TestProvider_UpdateRecordsById(t *testing.T) {
	provider, zone := getProvider(t)

	//Get records
	initialRecords := getInitialRecords(t, provider, zone)

	ttl := time.Duration(600 * time.Second)
	testFullName := "libdns_test_record"

	//Create record
	_, err := createOrGetTestRecord(t, provider, zone)
	if err != nil {
		t.Error(err)
		return
	}

	updatedTestValue := "updated-test-value"
	// Update record
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{
		libdns.TXT{
			Name: testFullName,
			TTL:  ttl,
			Text: updatedTestValue,
		},
	})

	if err != nil {
		t.Error(err)
	}

	if len(updatedRecords) != 1 {
		t.Logf("Incorrect amount of records changed")
	}

	t.Logf("Updated record: \n%v\n", updatedRecords[0])

	//Get records
	postUpdatedRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Error(err)
	}

	if len(postUpdatedRecords) != len(initialRecords)+1 {
		t.Errorf("Additional record created instead of updating existing. Started with: %d, now has: %d", len(initialRecords), len(postUpdatedRecords))
	}
}

func TestProvider_UpdateRecordsByLookup(t *testing.T) {
	provider, zone := getProvider(t)

	//Get records
	initialRecords := getInitialRecords(t, provider, zone)

	ttl := time.Duration(600 * time.Second)
	testFullName := "libdns_test_record"

	//Create record
	_, err := createOrGetTestRecord(t, provider, zone)
	if err != nil {
		t.Error(err)
		return
	}

	updatedTestValue := "updated-test-value-by-lookup"
	// Update record
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{
		libdns.TXT{
			Name: testFullName,
			TTL:  ttl,
			Text: updatedTestValue,
		},
	})

	if err != nil {
		t.Error(err)
	}

	if len(updatedRecords) != 1 {
		t.Logf("Incorrect amount of records changed")
	}

	t.Logf("Updated record: \n%v\n", updatedRecords[0])

	//Get records
	postUpdatedRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Error(err)
	}

	if len(postUpdatedRecords) != len(initialRecords)+1 {
		t.Errorf("Additional record created instead of updating existing. Started with: %d, now has: %d", len(initialRecords), len(postUpdatedRecords))
	}
}

func TestProvider_DeleteRecords(t *testing.T) {
	provider, zone := getProvider(t)

	//Create record
	createdRecord, err := createOrGetTestRecord(t, provider, zone)

	if err != nil {
		t.Error(err)
		return
	}

	// Delete record
	deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{*createdRecord})

	if err != nil {
		t.Error(err)
	}

	if len(deleteRecords) != 1 {
		t.Errorf("Deleted incorrect amount of records %d", len(deleteRecords))
	}

	t.Logf("Deleted record: \n%v\n", deleteRecords[0])
}
