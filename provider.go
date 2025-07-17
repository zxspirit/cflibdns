package provider

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/libdns/libdns"
	"os"
	"time"
)

// New creates a new Provider instance with a Cloudflare client and caches for zones and records.
func New() *Provider {
	return &Provider{
		client:      cloudflare.NewClient(option.WithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))),
		zoneCache:   make(map[string]string),
		recordCache: make(map[string]string),
	}
}

type Provider struct {
	client      *cloudflare.Client
	zoneCache   map[string]string
	recordCache map[string]string
}

func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	res, err := p.client.Zones.List(ctx, zones.ZoneListParams{
		Status: cloudflare.F(zones.ZoneListParamsStatusActive),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing zones: %w", err)
	}
	z := make([]libdns.Zone, 0, len(res.Result))
	for _, zone := range res.Result {
		z = append(z, libdns.Zone{
			Name: zone.Name,
		})
		p.zoneCache[zone.Name] = zone.ID
		p.zoneCache[zone.Name+"."] = zone.ID
	}
	return z, nil
}

func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	for _, rec := range recs {
		rr := rec.RR()
		s, ok := p.recordCache[rr.Name]
		if !ok {
			return nil, fmt.Errorf("record not found: %s", rr.Name)
		}
		_, err := p.client.DNS.Records.Delete(ctx, s, dns.RecordDeleteParams{ZoneID: cloudflare.F(zone)})
		if err != nil {
			return nil, fmt.Errorf("error deleting record %s: %w", rr.Name, err)
		}
		// Remove from record cache
		delete(p.recordCache, rr.Name)

	}
	return recs, nil
}

func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	for _, rec := range recs {
		rr := rec.RR()
		zoneId, ok := p.zoneCache[zone]
		if !ok {
			_, err := p.ListZones(ctx)
			if err != nil {
				return nil, fmt.Errorf("error listing zones: %w", err)
			}
			zoneId, ok = p.zoneCache[zone]
			if !ok {
				return nil, fmt.Errorf("zone not found: %s", zone)
			}
		}
		res, err := p.client.DNS.Records.List(ctx, dns.RecordListParams{
			ZoneID: cloudflare.F(zoneId),
			Name: cloudflare.F(dns.RecordListParamsName{
				Exact: cloudflare.F(rr.Name),
			}),
			Type: cloudflare.F(dns.RecordListParamsType(rr.Type)),
		})
		if err != nil {
			return nil, fmt.Errorf("error listing records for zone %s zoneId: %w", zone, err)
		}
		switch len(res.Result) {
		case 0:
			if rr.Data == "" {
				continue
			}
			body, err := p.getParam(rr)
			if err != nil {
				return nil, fmt.Errorf("error creating new record param for %s: %w", rr.Name, err)
			}
			response, err := p.client.DNS.Records.New(ctx, dns.RecordNewParams{
				ZoneID: cloudflare.F(zoneId),
				Body:   body.(dns.RecordNewParamsBodyUnion),
			})
			if err != nil {
				return nil, fmt.Errorf("error creating record %s zoneId in zone %s zoneId: %w", rr.Name, zone, err)
			}
			p.recordCache[rr.Name] = response.ID
		case 1:
			if rr.Data == "" {
				_, err := p.DeleteRecords(ctx, zone, recs)
				if err != nil {
					return nil, fmt.Errorf("error deleting record %s zoneId in zone %s zoneId: %w", rr.Name, zone, err)
				}
				continue
			}
			body, err := p.getParam(rr)
			if err != nil {
				return nil, fmt.Errorf("error creating update record param for %s: %w", rr.Name, err)
			}
			update, err := p.client.DNS.Records.Update(ctx, res.Result[0].ID, dns.RecordUpdateParams{
				ZoneID: cloudflare.F(zoneId),
				Body:   body.(dns.RecordUpdateParamsBodyUnion),
			})
			if err != nil {
				return nil, fmt.Errorf("error updating record %s zoneId in zone %s zoneId: %w", rr.Name, zone, err)
			}
			p.recordCache[rr.Name] = update.ID
		default:
			return nil, fmt.Errorf("multiple records found for %s zoneId in zone %s zoneId", rr.Name, zone)

		}

	}
	return recs, nil
}

func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {

	if zi, ok := p.zoneCache[zone]; ok {
		res, err := p.client.DNS.Records.List(ctx, dns.RecordListParams{ZoneID: cloudflare.F(zi)})
		if err != nil {
			return nil, fmt.Errorf("error listing records for zone %s: %w", zone, err)
		}
		records := make([]libdns.Record, 0, len(res.Result))
		for _, rec := range res.Result {

			records = append(records, libdns.RR{
				Name: rec.Name,
				TTL:  time.Duration(rec.TTL) * time.Second,
				Type: string(rec.Type),
				Data: rec.Content,
			})
			p.recordCache[rec.Name] = rec.ID
		}
		return records, nil
	} else {
		return nil, fmt.Errorf("zone not found: %s", zone)
	}
}

func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	for _, rec := range recs {
		s, ok := p.zoneCache[zone]
		if !ok {
			_, err := p.ListZones(ctx)
			if err != nil {
				return nil, fmt.Errorf("error listing zones: %w", err)
			}
			s, ok = p.zoneCache[zone]
			if !ok {
				return nil, fmt.Errorf("zone not found: %s", zone)
			}
		}
		body, e := p.getParam(rec)
		rr := rec.RR()
		if e != nil {
			return nil, fmt.Errorf("error creating new record param for %s: %w", rr.Name, e)
		}
		rrr, err := p.client.DNS.Records.New(ctx, dns.RecordNewParams{
			ZoneID: cloudflare.F(s),
			Body:   body.(dns.RecordNewParamsBodyUnion),
		})
		p.recordCache[rr.Name] = rrr.ID
		if err != nil {
			return nil, fmt.Errorf("error creating record %s: %w", rr.Name, err)
		}
	}
	return recs, nil
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
