package provider

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

func formatDomain(domain string) string {
	domain = strings.ToLower(domain)
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}
	return domain
}

type cache struct {
	zones []*zone
	mu    sync.RWMutex
}

func (c *cache) addZone(z *zone) {
	c.mu.Lock()
	defer c.mu.Unlock()
	z.name = formatDomain(z.name)
	c.zones = append(c.zones, z)
}
func (c *cache) getAllZones() []*zone {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.zones
}
func (c *cache) getZone(name string) (*zone, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, z := range c.zones {
		if z.name == name {
			return z, nil
		}
	}
	return nil, fmt.Errorf("zone %s not found", name)
}

type zone struct {
	id      string
	name    string
	records []*record
	mu      sync.RWMutex
}

func (z *zone) getAllRecords() []*record {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.records
}

func (z *zone) addRecord(record *record) {
	z.mu.Lock()
	defer z.mu.Unlock()
	record.name = formatDomain(record.name)
	z.records = append(z.records, record)
}
func (z *zone) getRecord(name string, recordType string) (*record, error) {
	z.mu.RLock()
	defer z.mu.RUnlock()
	for _, rec := range z.records {
		if rec.name == name && rec.dnsType == recordType {
			return rec, nil
		}
	}
	return nil, fmt.Errorf("record %s not found", name)
}
func (z *zone) deleteRecord(id string) error {
	z.mu.Lock()
	defer z.mu.Unlock()
	for i, r := range z.records {
		if r.id == id {
			z.records = append(z.records[:i], z.records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("record %s not found", id)
}
func (z *zone) updateRecordById(record *record) error {
	z.mu.Lock()
	defer z.mu.Unlock()
	for i, r := range z.records {
		if r.id == record.id {
			z.records[i] = record
			return nil
		}
	}
	return fmt.Errorf("record %s not found", record.id)
}

type record struct {
	id      string
	content string
	name    string
	dnsType string
	ttl     time.Duration
}
