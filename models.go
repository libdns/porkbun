package porkbun

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

type pkbnRecord struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Notes   string `json:"notes"`
	Prio    string `json:"prio"`
	TTL     string `json:"ttl"`
	Type    string `json:"type"`
}

type pkbnRecordsResponse struct {
	Records []pkbnRecord `json:"records"`
	Status  string       `json:"status"`
}

type ApiCredentials struct {
	Apikey       string `json:"apikey"`
	Secretapikey string `json:"secretapikey"`
}

type pkbnResponseStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
type pkbnPingResponse struct {
	pkbnResponseStatus
	YourIP string `json:"yourIp"`
}

type pkbnCreateResponse struct {
	pkbnResponseStatus
}

func (record pkbnRecord) toLibdnsRecord(zone string) (libdns.Record, error) {
	name := libdns.RelativeName(record.Name, zone)
	ttl, err := time.ParseDuration(record.TTL + "s")
	if err != nil {
		return libdns.RR{}, err
	}

	switch record.Type {
	case "A", "AAAA":
		ip, err := netip.ParseAddr(record.Content)
		if err != nil {
			return libdns.Address{}, err
		}
		return libdns.Address{
			Name: name,
			TTL:  ttl,
			IP:   ip,
		}, nil
	case "CAA":
		contentParts := strings.SplitN(record.Content, " ", 3)
		flags, err := strconv.Atoi(contentParts[0])
		if err != nil {
			return libdns.CAA{}, err
		}
		tag := contentParts[1]
		value := contentParts[2]
		return libdns.CAA{
			Name:  name,
			TTL:   ttl,
			Flags: uint8(flags),
			Tag:   tag,
			Value: value,
		}, nil
	case "CNAME":
		return libdns.CNAME{
			Name:   name,
			TTL:    ttl,
			Target: record.Content,
		}, nil
	case "SRV":
		priority, err := strconv.Atoi(record.Prio)
		return libdns.SRV{
			Service:   "",
			Transport: "",
			Name:      name,
			TTL:       ttl,
			Priority:  uint16(priority),
			Weight:    0,
			Port:      0,
			Target:    "",
		}, err
	case "TXT":
		return libdns.TXT{
			Name: name,
			TTL:  ttl,
			Text: record.Content,
		}, err
	default:
		return libdns.RR{}, fmt.Errorf("Unsupported record type: %s", record.Type)
	}
}

func (a pkbnResponseStatus) Error() string {
	return fmt.Sprintf("%s: %s", a.Status, a.Message)
}

type pkbnRecordPayload struct {
	*ApiCredentials
	Content string `json:"content"`
	Name    string `json:"name"`
	TTL     string `json:"ttl"`
	Type    string `json:"type"`
}
