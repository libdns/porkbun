package porkbun

import (
	"fmt"
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func TestPorkbunRecord_ToLibdnsRecord(t *testing.T) {
	testCases := []struct {
		porkbunRecord pkbnRecord
		want          libdns.Record
	}{
		// root A record
		{
			pkbnRecord{
				Content: "1.1.1.1",
				Name:    "example.com",
				Notes:   "",
				Prio:    "0",
				TTL:     "600",
				Type:    "A",
			},
			libdns.Address{
				Name: "@",
				TTL:  mustParseDuration("10m"),
				IP:   netip.MustParseAddr("1.1.1.1"),
			},
		},
		// subdomain A record
		{
			pkbnRecord{
				Content: "1.1.1.2",
				Name:    "test.example.com",
				Notes:   "",
				Prio:    "0",
				TTL:     "300",
				Type:    "A",
			},
			libdns.Address{
				Name: "test",
				TTL:  mustParseDuration("5m"),
				IP:   netip.MustParseAddr("1.1.1.2"),
			},
		},
		// subdomain CNAME record
		{
			pkbnRecord{
				Content: "target.example.com.",
				Name:    "test.example.com",
				Notes:   "",
				Prio:    "0",
				TTL:     "300",
				Type:    "CNAME",
			},
			libdns.CNAME{
				Name:   "test",
				TTL:    mustParseDuration("5m"),
				Target: "target.example.com.",
			},
		},
		// SRV record
		{
			pkbnRecord{
				Content: "1 993 imap.example.com",
				Name:    "_imaps._tcp.example.com",
				Notes:   "",
				Prio:    "10",
				TTL:     "300",
				Type:    "SRV",
			},
			libdns.SRV{
				Service:   "imaps",
				Transport: "tcp",
				Name:      "example.com",
				TTL:       mustParseDuration("5m"),
				Priority:  10,
				Weight:    1,
				Port:      993,
				Target:    "imap.example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.porkbunRecord.Type, tc.want.RR().Name), func(t *testing.T) {
			var err error
			switch tc.want.(type) {
			case libdns.Address:
				err = equalsAddress(tc.porkbunRecord, tc.want.(libdns.Address))
			case libdns.CNAME:
				err = equalsCNAME(tc.porkbunRecord, tc.want.(libdns.CNAME))
			case libdns.SRV:
				err = equalsSRV(tc.porkbunRecord, tc.want.(libdns.SRV))
			default:
				err = fmt.Errorf("unhandled record type: %s", tc.porkbunRecord.Type)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func equalsAddress(porkbunRecord pkbnRecord, want libdns.Address) error {
	libdnsRecord, err := porkbunRecord.toLibdnsRecord("example.com")
	if err != nil {
		return err
	}

	address, ok := libdnsRecord.(libdns.Address)
	if !ok {
		return fmt.Errorf("invalid type returned. wanted libdns.Address, got %v", reflect.TypeOf(libdnsRecord))
	}

	if address.Name != want.Name {
		return fmt.Errorf("incorrect name. wanted '%s' got '%s'", want.Name, address.Name)
	}

	if address.TTL != want.TTL {
		return fmt.Errorf("incorrect TTL. wanted '%v' got '%v'", want.TTL, address.TTL)
	}

	if address.IP.String() != want.IP.String() {
		return fmt.Errorf("incorrect IP. wanted '%v' got '%v'", want.IP, address.IP)
	}

	return nil
}

func equalsCNAME(porkbunRecord pkbnRecord, want libdns.CNAME) error {
	libdnsRecord, err := porkbunRecord.toLibdnsRecord("example.com")
	if err != nil {
		return err
	}

	cname, ok := libdnsRecord.(libdns.CNAME)
	if !ok {
		return fmt.Errorf("invalid type returned. wanted libdns.CNAME, got %v", reflect.TypeOf(libdnsRecord))
	}

	if cname.Name != want.Name {
		return fmt.Errorf("incorrect name. wanted '%s' got '%s'", want.Name, cname.Name)
	}

	if cname.TTL != want.TTL {
		return fmt.Errorf("incorrect TTL. wanted '%v' got '%v'", want.TTL, cname.TTL)
	}

	if cname.Target != want.Target {
		return fmt.Errorf("incorrect Target. wanted '%v' got '%v'", want.Target, cname.Target)
	}

	return nil
}

func equalsSRV(porkbunRecord pkbnRecord, want libdns.SRV) error {
	libdnsRecord, err := porkbunRecord.toLibdnsRecord("example.com")
	if err != nil {
		return err
	}

	srv, ok := libdnsRecord.(libdns.SRV)
	if !ok {
		return fmt.Errorf("invalid type returned. wanted libdns.SRV, got %v", reflect.TypeOf(libdnsRecord))
	}

	if srv.Service != want.Service {
		return fmt.Errorf("incorrect Service. wanted '%s' got '%s'", want.Service, srv.Service)
	}

	if srv.Transport != want.Transport {
		return fmt.Errorf("incorrect Transport. wanted '%s' got '%s'", want.Transport, srv.Transport)
	}

	if srv.Name != want.Name {
		return fmt.Errorf("incorrect name. wanted '%s' got '%s'", want.Name, srv.Name)
	}

	if srv.TTL != want.TTL {
		return fmt.Errorf("incorrect TTL. wanted '%v' got '%v'", want.TTL, srv.TTL)
	}

	if srv.Priority != want.Priority {
		return fmt.Errorf("incorrect Priority. wanted '%v' got '%v'", want.Priority, srv.Priority)
	}

	if srv.Weight != want.Weight {
		return fmt.Errorf("incorrect Weight. wanted '%v' got '%v'", want.Weight, srv.Weight)
	}

	if srv.Port != want.Port {
		return fmt.Errorf("incorrect Port. wanted '%v' got '%v'", want.Port, srv.Port)
	}

	if srv.Target != want.Target {
		return fmt.Errorf("incorrect Target. wanted '%v' got '%v'", want.Target, srv.Target)
	}

	return nil
}

func mustParseDuration(durationStr string) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		panic(err)
	}
	return duration
}
