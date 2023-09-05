package porkbun

import "fmt"

type PorkbunRecord struct {
	Content string `json:"content"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Notes   string `json:"notes"`
	Prio    string `json:"prio"`
	TTL     string `json:"ttl"`
	Type    string `json:"type"`
}

type ApiRecordsResponse struct {
	Records []PorkbunRecord `json:"records"`
	Status  string          `json:"status"`
}

type ApiCredentials struct {
	Apikey       string `json:"apikey"`
	Secretapikey string `json:"secretapikey"`
}

type ResponseStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
type PingResponse struct {
	ResponseStatus
	YourIP string `json:"yourIp"`
}

func (a ResponseStatus) Error() string {
	return fmt.Sprintf("%s: %s", a.Status, a.Message)
}

type RecordCreateRequest struct {
	*ApiCredentials
	Content string `json:"content"`
	Name    string `json:"name"`
	TTL     string `json:"ttl"`
	Type    string `json:"type"`
}

type RecordUpdateRequest struct {
	*ApiCredentials
	Content string `json:"content"`
	TTL     string `json:"ttl"`
}

type Set struct {
	m map[string]bool
}

func NewSet() *Set {
	s := &Set{}
	s.m = make(map[string]bool)
	return s
}

func (s *Set) Add(value string) {
	s.m[value] = true
}

func (s *Set) Remove(value string) {
	delete(s.m, value)
}

func (s *Set) Contains(value string) bool {
	_, c := s.m[value]
	return c
}
