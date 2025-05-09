package porkbun

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/libdns/libdns"
)

func updateRecordTTL(record libdns.Record, newTTL time.Duration) (libdns.Record, error) {
	// Create a copy of the record with new TTL
	switch r := record.(type) {
	case libdns.TXT:
		return libdns.TXT{
			Name: r.Name,
			TTL:  newTTL,
			Text: r.Text,
		}, nil
	case libdns.Address:
		return libdns.Address{
			Name: r.Name,
			TTL:  newTTL,
			IP:   r.IP,
		}, nil
	}

	return nil, fmt.Errorf("unsupported DNS record type: %T", record)

}

func getProvider(t *testing.T) (Provider, string) {
	envErr := godotenv.Load()
	if envErr != nil {
		t.Fatal(envErr)
	}

	apikey := os.Getenv("PORKBUN_API_KEY")
	secretApiKey := os.Getenv("PORKBUN_SECRET_API_KEY")
	zone := os.Getenv("ZONE")

	if apikey == "" || secretApiKey == "" || zone == "" {
		t.Fatal("All variables must be set in '.env' file")
	}

	provider := Provider{
		APIKey:       apikey,
		APISecretKey: secretApiKey,
	}

	return provider, zone
}

func TestProvider_CheckCredentials(t *testing.T) {
	provider, _ := getProvider(t)
	//Check Authorization
	_, err := provider.CheckCredentials(context.TODO())

	if err != nil {
		t.Fatal(err)
	}
}

func TestProvider_GetRecords(t *testing.T) {
	provider, zone := getProvider(t)
	records, err := provider.GetRecords(context.TODO(), zone)

	if err != nil {
		t.Fatal(err)
	}

	log.Println("Records fetched:")
	for _, record := range records {
		rr := record.RR()
		t.Logf("%s (.%s): %s, %s\n", rr.Name, zone, rr.Data, rr.Type)
	}
}

func TestProvider_SetRecords(t *testing.T) {
	provider, zone := getProvider(t)
	modifyRoot := os.Getenv("MODIFY_ROOT") == "true"

	initialTtl := 600 * time.Second
	expectedTtl := 900 * time.Second
	testCases := []libdns.Record{
		libdns.TXT{
			Name: "libdns.set-sub",
			TTL:  initialTtl,
			Text: "libdns_test_modify_sub_txt",
		},
		libdns.Address{
			Name: "libdns-set-address",
			TTL:  initialTtl,
			IP:   netip.MustParseAddr("1.1.1.1"),
		},
		libdns.TXT{
			Name: "libdns.set-sub-new",
			TTL:  initialTtl,
			Text: "libdns_test_modify_sub_new_txt",
		},
		libdns.Address{
			Name: "libdns-set-address-new",
			TTL:  initialTtl,
			IP:   netip.MustParseAddr("1.1.1.1"),
		},
	}

	if modifyRoot {
		testCases = append(testCases, libdns.TXT{
			Name: "@",
			TTL:  initialTtl,
			Text: "libdns_test_modify_root_txt",
		})
	} else {
		t.Log("Skipping root record modification")
	}

	_, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{testCases[0], testCases[1]})

	if err != nil {
		t.Fatal(err)
	}
	for index, expectedRecord := range testCases {
		t.Run(expectedRecord.RR().Type+" "+expectedRecord.RR().Name, func(t *testing.T) {
			if index > 0 {
				time.Sleep(1 * time.Second)
			}

			updatedRecord, err := updateRecordTTL(expectedRecord, expectedTtl)

			if err != nil {
				t.Fatal(err)
			}

			records, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{updatedRecord})
			if err != nil {
				t.Fatal(err)
			}

			if len(records) != 1 {
				t.Fatal("Incorrect amount of records created")
			}

			if records[0].RR().TTL != expectedTtl {
				t.Fatalf("Incorrect TTL. Wanted %v, got %v", expectedTtl, records[0].RR().TTL)
			}

			t.Logf("Set record ttl")
		})
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	provider, zone := getProvider(t)
	modifyRoot := os.Getenv("MODIFY_ROOT") == "true"

	testCases := []libdns.Record{
		libdns.TXT{
			Name: "libdns.sub-append",
			TTL:  600 * time.Second,
			Text: "libdns_test_append_sub_txt",
		},
		libdns.Address{
			Name: "libdns-address-append",
			TTL:  600 * time.Second,
			IP:   netip.MustParseAddr("1.1.1.1"),
		},
	}

	if modifyRoot {
		testCases = append(testCases, libdns.TXT{
			Name: "@",
			TTL:  600 * time.Second,
			Text: "libdns_test_append_root_txt",
		})
	} else {
		t.Log("Skipping root record modification")
	}

	// Delete records before modifying them.
	_, _ = provider.DeleteRecords(context.TODO(), zone, testCases)

	for index, expectedRecord := range testCases {
		t.Run(expectedRecord.RR().Type, func(t *testing.T) {
			if index > 0 {
				time.Sleep(1 * time.Second)
			}

			records, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{expectedRecord})
			if err != nil {
				t.Fatal(err)
			}

			if len(records) != 1 {
				t.Fatal("Incorrect amount of records appended")
			}

			t.Logf("Appended record: \n%v\n", records[0])
		})
	}

	_, _ = provider.DeleteRecords(context.TODO(), zone, testCases)
}

func TestProvider_DeleteRecords(t *testing.T) {
	provider, zone := getProvider(t)

	expectedRecord := libdns.TXT{
		Name: "libdns_delete",
		TTL:  600 * time.Second,
		Text: "libdns_test_delete",
	}

	records, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{expectedRecord})
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 1 {
		t.Fatal("Incorrect amount of records created")
	}

	t.Logf("Created record: \n%v\n", records[0])

	// Delete record
	deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{expectedRecord})

	if err != nil {
		t.Fatal(err)
	}

	if len(deleteRecords) != 1 {
		t.Fatalf("Deleted incorrect amount of records %d", len(deleteRecords))
	}

	t.Logf("Deleted record: \n%v\n", deleteRecords[0])
}
