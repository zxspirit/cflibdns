package provider

import (
	"fmt"
	"strings"
	"time"
)

type cache struct {
	zones []*zone
}

func (c *cache) addZone(z *zone) {
	z.name = strings.ToLower(z.name)
	if strings.HasSuffix(z.name, ".") {
		z.name = z.name[:len(z.name)-1]
	}
	c.zones = append(c.zones, z)
}
func (c *cache) getAllZones() []*zone {
	return c.zones
}
func (c *cache) getZone(name string) (*zone, error) {
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
}

func (z *zone) getAllRecords() []*record {
	return z.records
}

func (z *zone) addRecord(record *record) {

	record.name = strings.ToLower(record.name)
	if strings.HasSuffix(record.name, ".") {
		record.name = record.name[:len(record.name)-1]
	}
	z.records = append(z.records, record)
}
func (z *zone) getRecord(name string, recordType string) (*record, error) {
	for _, rec := range z.records {
		if rec.name == name && rec.dnsType == recordType {
			return rec, nil
		}
	}
	return nil, fmt.Errorf("record %s not found", name)
}
func (z *zone) deleteRecord(id string) error {
	for i, r := range z.records {
		if r.id == id {
			z.records = append(z.records[:i], z.records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("record %s not found", id)
}
func (z *zone) updateRecordById(record *record) error {
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
