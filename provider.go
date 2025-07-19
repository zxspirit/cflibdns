package cflibdns

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/libdns/libdns"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

// New creates a new Provider instance with a Cloudflare client and caches for zones and records.
func New(logger *logrus.Logger) *Provider {

	return &Provider{
		client: cloudflare.NewClient(option.WithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))),
		cache:  cache{zones: make([]*zone, 0)},
		logger: logger,
	}
}

type Provider struct {
	client *cloudflare.Client
	cache  cache
	logger *logrus.Logger
}

func (p *Provider) InitCache(ctx context.Context) error {
	res, err := p.client.Zones.List(ctx, zones.ZoneListParams{Status: cloudflare.F(zones.ZoneListParamsStatusActive)})
	if err != nil {
		return fmt.Errorf("error listing zones: %w", err)
	}
	zs := make([]*zone, 0, len(res.Result))
	for _, z := range res.Result {
		zs = append(zs, &zone{
			id:      z.ID,
			name:    z.Name,
			records: make([]*record, 0),
		})
	}
	p.cache.setZones(zs)
	for _, z := range zs {
		list, err := p.client.DNS.Records.List(ctx, dns.RecordListParams{
			ZoneID: cloudflare.F(z.id),
		})
		if err != nil {
			return fmt.Errorf("error listing records for zone %s: %w", z.name, err)
		}
		rs := make([]*record, 0, len(list.Result))

		for _, rec := range list.Result {
			rs = append(rs, &record{
				id:      rec.ID,
				content: rec.Content,
				name:    rec.Name,
				dnsType: string(rec.Type),
				ttl:     time.Duration(rec.TTL),
			})
		}
		z.setRecords(rs)
	}
	p.logger.Info("cache initialized with zones and records")
	return nil
}

func (p *Provider) ListZones(_ context.Context) ([]libdns.Zone, error) {
	allZones := p.cache.getAllZones()
	z := make([]libdns.Zone, 0, len(p.cache.zones))
	for _, zc := range allZones {
		z = append(z, libdns.Zone{Name: zc.name})
	}
	p.logger.Info("zones retrieved: ", z)
	return z, nil
}

func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	getZone, err := p.cache.getZone(zone)
	if err != nil {
		return nil, fmt.Errorf("error getting zone %s: %w", zone, err)
	}
	r := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		getRecord, err := getZone.getRecord(rec.RR().Name, rec.RR().Type)
		if err != nil {
			return nil, fmt.Errorf("error getting record %s: %w", rec.RR().Name, err)
		}
		res, err := p.client.DNS.Records.Delete(ctx, getRecord.id, dns.RecordDeleteParams{ZoneID: cloudflare.F(getZone.id)})
		if err != nil {
			return nil, fmt.Errorf("error deleting record %s: %w", rec.RR().Name, err)
		}
		err = getZone.deleteRecord(res.ID)
		if err != nil {
			return nil, fmt.Errorf("error deleting record %s from cache: %w", rec.RR().Name, err)
		}
		r = append(r, libdns.RR{
			Name: rec.RR().Name,
			TTL:  rec.RR().TTL,
			Type: rec.RR().Type,
			Data: rec.RR().Data,
		})
	}
	p.logger.Infof("deleted records: %v", r)
	return r, nil
}

func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	zoneCache, err := p.cache.getZone(zone)
	if err != nil {
		return nil, fmt.Errorf("error getting zone %s: %w", zone, err)

	}
	r := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		param, err := p.getParam(rec)
		if err != nil {
			return nil, fmt.Errorf("error getting parameters for record %s", rec.RR().Name)
		}
		recordCache, err := zoneCache.getRecord(rec.RR().Name, rec.RR().Type)
		if err != nil {
			if rec.RR().Data == "" {
				continue // Skip if the record data is empty
			}
			// Record does not exist, create it
			union, ok := param.(dns.RecordNewParamsBodyUnion)
			if !ok {
				return nil, fmt.Errorf("error casting parameters for record %s: %w", rec.RR().Name, err)
			}
			res, err := p.client.DNS.Records.New(ctx, dns.RecordNewParams{
				ZoneID: cloudflare.F(zoneCache.id),
				Body:   union,
			})
			if err != nil {
				return nil, fmt.Errorf("error creating record %s: %w", rec.RR().Name, err)
			}
			zoneCache.addRecord(&record{
				id:      res.ID,
				content: res.Content,
				name:    res.Name,
				dnsType: string(res.Type),
				ttl:     time.Duration(res.TTL),
			})
			r = append(r, libdns.RR{
				Name: res.Name,
				TTL:  time.Duration(res.TTL),
				Type: string(res.Type),
				Data: res.Content,
			})
		} else {
			if rec.RR().Data == "" {
				// If the record exists but data is empty, delete it
				res, err := p.client.DNS.Records.Delete(ctx, recordCache.id, dns.RecordDeleteParams{ZoneID: cloudflare.F(zoneCache.id)})
				if err != nil {
					return nil, fmt.Errorf("error deleting record %s: %w", rec.RR().Name, err)
				}
				err = zoneCache.deleteRecord(res.ID)
				if err != nil {
					return nil, fmt.Errorf("error deleting record %s from cache: %w", rec.RR().Name, err)
				}
				p.logger.Infof("deleted record %s as data is empty", rec.RR().Name)
				continue // Skip to the next record
			}
			res, err := p.client.DNS.Records.Update(ctx, recordCache.id, dns.RecordUpdateParams{
				ZoneID: cloudflare.F(zoneCache.id),
				Body:   param.(dns.RecordUpdateParamsBodyUnion),
			})
			if err != nil {
				return nil, fmt.Errorf("error updating record %s: %w", rec.RR().Name, err)
			}
			err = zoneCache.updateRecordById(&record{
				id:      res.ID,
				content: res.Content,
				name:    res.Name,
				dnsType: string(res.Type),
				ttl:     time.Duration(res.TTL),
			})
			if err != nil {
				return nil, fmt.Errorf("error updating record %s in cache: %w", rec.RR().Name, err)
			}
			r = append(r, libdns.RR{
				Name: res.Name,
				TTL:  time.Duration(res.TTL),
				Type: string(res.Type),
				Data: res.Content,
			})
			p.logger.Infof("updated record %s", rec.RR().Name)
		}
	}
	p.logger.Infof("set records: %v", r)
	return r, nil
}

func (p *Provider) GetRecords(_ context.Context, zone string) ([]libdns.Record, error) {
	getZone, err := p.cache.getZone(zone)
	if err != nil {
		return nil, fmt.Errorf("error getting zone %s: %w", zone, err)
	}
	records := getZone.getAllRecords()
	recs := make([]libdns.Record, 0, len(records))
	for _, r := range records {
		recs = append(recs, libdns.RR{
			Name: r.name,
			TTL:  r.ttl,
			Type: r.dnsType,
			Data: r.content,
		})

	}
	p.logger.Infof("records retrieved for zone %s: %v", zone, recs)
	return recs, nil
}

func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	zoneCache, err := p.cache.getZone(zone)
	if err != nil {
		return nil, fmt.Errorf("error getting zone %s: %w", zone, err)
	}
	r := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		_, err := zoneCache.getRecord(rec.RR().Name, rec.RR().Type)
		if err == nil {
			return nil, fmt.Errorf("record %s already exists in zone %s", rec.RR().Name, zone)

		}
		param, err := p.getParam(rec)
		if err != nil {
			return nil, fmt.Errorf("error getting parameters for record %s: %w", rec.RR().Name, err)
		}
		union, ok := param.(dns.RecordNewParamsBodyUnion)
		if !ok {
			return nil, fmt.Errorf("error casting parameters for record %s", rec.RR().Name)
		}
		res, err := p.client.DNS.Records.New(ctx, dns.RecordNewParams{
			ZoneID: cloudflare.F(zoneCache.id),
			Body:   union,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating record %s: %w", rec.RR().Name, err)
		}
		zoneCache.addRecord(&record{
			id:      res.ID,
			content: res.Content,
			name:    res.Name,
			dnsType: string(res.Type),
			ttl:     time.Duration(res.TTL),
		})
		r = append(r, libdns.RR{
			Name: res.Name,
			TTL:  time.Duration(res.TTL),
			Type: string(res.Type),
			Data: res.Content,
		})

	}
	p.logger.Infof("appended records: %v", r)
	return r, nil

}

func (p *Provider) getParam(r libdns.Record) (interface{}, error) {
	switch r.RR().Type {
	case "A":
		return &dns.ARecordParam{
			Name:    cloudflare.F(r.RR().Name),
			TTL:     cloudflare.F(dns.TTL(r.RR().TTL)),
			Type:    cloudflare.F(dns.ARecordType(r.RR().Type)),
			Content: cloudflare.F(r.RR().Data),
		}, nil
	case "AAAA":
		return &dns.AAAARecordParam{Name: cloudflare.F(r.RR().Name),
			TTL:     cloudflare.F(dns.TTL(r.RR().TTL)),
			Type:    cloudflare.F(dns.AAAARecordType(r.RR().Type)),
			Content: cloudflare.F(r.RR().Data)}, nil
	case "CNAME":
		return &dns.CNAMERecordParam{Name: cloudflare.F(r.RR().Name),
			TTL:     cloudflare.F(dns.TTL(r.RR().TTL)),
			Type:    cloudflare.F(dns.CNAMERecordType(r.RR().Type)),
			Content: cloudflare.F(r.RR().Data)}, nil
	case "TXT":
		return &dns.TXTRecordParam{Name: cloudflare.F(r.RR().Name),
			TTL:     cloudflare.F(dns.TTL(r.RR().TTL)),
			Type:    cloudflare.F(dns.TXTRecordType(r.RR().Type)),
			Content: cloudflare.F(r.RR().Data)}, nil
	// 可根据需要继续添加其他类型
	default:
		return nil, fmt.Errorf("unsupported record type: %s", r.RR().Type)
	}
}

var (
	_ libdns.ZoneLister    = (*Provider)(nil)
	_ libdns.RecordDeleter = (*Provider)(nil)
	_ libdns.RecordSetter  = (*Provider)(nil)
	_ libdns.RecordGetter  = (*Provider)(nil)
)
