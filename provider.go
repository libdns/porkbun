// Package porkbun implements a DNS record management client compatible
// with the libdns interfaces for Porkbun.
package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Porkbun.
type Provider struct {
	APIKey       string `json:"api_key,omitempty"`
	APISecretKey string `json:"api_secret_key,omitempty"`
}

func (p *Provider) getApiHost() string {
	return "https://porkbun.com/api/json/v3/"
}

func (p *Provider) getRecordCoordinates(record libdns.Record) string {
	return fmt.Sprintf("%s-%s", record.Name, record.Type)
}

func (p *Provider) getCredentials() ApiCredentials {
	return ApiCredentials{p.APIKey, p.APISecretKey}
}

// Strips the trailing dot from a Zone
func trimZone(zone string) string {
	return strings.TrimSuffix(zone, ".")
}

func (p *Provider) CheckCredentials(_ context.Context) (string, error) {
	credentialJson, err := json.Marshal(p.getCredentials())
	if err != nil {
		return "", err
	}

	response, err := makeHttpRequest[PingResponse](p, "ping", bytes.NewReader(credentialJson), PingResponse{})

	if err != nil {
		return "", err
	}

	if response.Status != "SUCCESS" {
		return "", err
	}

	return response.YourIP, nil
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(_ context.Context, zone string) ([]libdns.Record, error) {
	trimmedZone := trimZone(zone)

	credentialJson, err := json.Marshal(p.getCredentials())
	if err != nil {
		return nil, err
	}
	response, err := makeHttpRequest[ApiRecordsResponse](p, "dns/retrieve/"+trimmedZone, bytes.NewReader(credentialJson), ApiRecordsResponse{})

	if err != nil {
		return nil, err
	}

	if response.Status != "SUCCESS" {
		return nil, errors.New(fmt.Sprintf("Invalid response status %s", response.Status))
	}

	var records []libdns.Record
	for _, record := range response.Records {
		ttl, err := time.ParseDuration(record.TTL + "s")
		if err != nil {
			return nil, err
		}
		priority, _ := strconv.Atoi(record.Prio)
		formatted := libdns.Record{
			ID:       record.ID,
			Name:     record.Name + ".",
			Priority: priority,
			TTL:      ttl,
			Type:     record.Type,
			Value:    record.Content,
		}
		records = append(records, formatted)
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(_ context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var createdRecords []libdns.Record

	for _, record := range records {
		if record.TTL/time.Second < 600 {
			record.TTL = 600 * time.Second
		}
		ttlInSeconds := int(record.TTL / time.Second)
		trimmedName := libdns.RelativeName(record.Name, zone)

		reqBody := RecordCreateRequest{&credentials, record.Value, trimmedName, strconv.Itoa(ttlInSeconds), record.Type}
		reqJson, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		response, err := makeHttpRequest(p, fmt.Sprintf("dns/create/%s", trimmedZone), bytes.NewReader(reqJson), ResponseStatus{})

		if err != nil {
			print(err)
			return nil, err
		}

		if response.Status != "SUCCESS" {
			return nil, errors.New(fmt.Sprintf("Invalid response status %s", response.Status))
		}
		createdRecords = append(createdRecords, record)
	}

	return createdRecords, nil
}

// UpdateRecords adds records to the zone. It returns the records that were added.
func (p *Provider) UpdateRecords(_ context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var createdRecords []libdns.Record

	for _, record := range records {
		if record.TTL/time.Second < 600 {
			record.TTL = 600 * time.Second
		}
		ttlInSeconds := int(record.TTL / time.Second)
		trimmedName := libdns.RelativeName(record.Name, zone)
		reqBody := RecordUpdateRequest{&credentials, record.Value, strconv.Itoa(ttlInSeconds)}
		reqJson, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		response, err := makeHttpRequest(p, fmt.Sprintf("dns/editByNameType/%s/%s/%s", trimmedZone, record.Type, trimmedName), bytes.NewReader(reqJson), ResponseStatus{})
		if err != nil {
			return nil, err
		}

		if response.Status != "SUCCESS" {
			return nil, err
		}
		createdRecords = append(createdRecords, record)
	}

	return createdRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	existingRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	existingCoordinates := NewSet()
	for _, r := range existingRecords {
		existingCoordinates.Add(p.getRecordCoordinates(r))
	}

	var updates []libdns.Record
	var creates []libdns.Record
	for _, r := range records {
		if existingCoordinates.Contains(p.getRecordCoordinates(r)) {
			updates = append(updates, r)
		} else {
			creates = append(creates, r)
		}
	}

	_, err = p.AppendRecords(ctx, zone, creates)
	if err != nil {
		return nil, err
	}
	_, err = p.UpdateRecords(ctx, zone, updates)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(_ context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var deletedRecords []libdns.Record

	for _, record := range records {
		reqJson, err := json.Marshal(credentials)
		if err != nil {
			return nil, err
		}
		trimmedName := libdns.RelativeName(record.Name, zone)

		_, err = makeHttpRequest(p, fmt.Sprintf("dns/deleteByNameType/%s/%s/%s", trimmedZone, record.Type, trimmedName), bytes.NewReader(reqJson), ResponseStatus{})
		if err != nil {
			return nil, err
		}
		deletedRecords = append(deletedRecords, record)
	}

	return deletedRecords, nil
}

func makeHttpRequest[T any](p *Provider, endpoint string, body io.Reader, responseType T) (T, error) {
	client := http.Client{}

	fullUrl := p.getApiHost() + endpoint
	u, err := url.Parse(fullUrl)
	if err != nil {
		return responseType, err
	}
	println(u.String())

	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		return responseType, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return responseType, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal("Couldn't close body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = errors.New("Invalid http response status, " + string(bodyBytes))
		return responseType, err
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return responseType, err
	}

	err = json.Unmarshal(result, &responseType)

	if err != nil {
		return responseType, err
	}

	return responseType, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
