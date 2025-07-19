package provider

import (
	"context"
	"github.com/libdns/libdns"
	"reflect"
	"testing"
)

func TestProvider_InitCache(t *testing.T) {
	provider := New()
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"t1", args{ctx: context.Background()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := provider.InitCache(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("InitCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProvider_ListZones(t *testing.T) {
	provider := New()
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []libdns.Zone
		wantErr bool
	}{
		{"t1", args{in0: context.Background()}, []libdns.Zone{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.ListZones(tt.args.in0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListZones() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListZones() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_SetRecords(t *testing.T) {
	provider := New()
	if err := provider.InitCache(context.Background()); err != nil {
		t.Fatalf("InitCache() error = %v", err)
	}

	type args struct {
		ctx  context.Context
		zone string
		recs []libdns.Record
	}
	tests := []struct {
		name    string
		args    args
		want    []libdns.Record
		wantErr bool
	}{
		{"t1", args{
			ctx:  context.Background(),
			zone: "newzhxu.com",
			recs: []libdns.Record{libdns.RR{
				Name: "test.newzhxu.com",
				TTL:  1,
				Type: "A",
				Data: "1.1.1.4",
			}},
		}, []libdns.Record{libdns.RR{
			Name: "test.newzhxu.com",
			TTL:  1,
			Type: "A",
			Data: "1.1.1.4",
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.SetRecords(tt.args.ctx, tt.args.zone, tt.args.recs)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetRecords() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	provider := New()
	if err := provider.InitCache(context.Background()); err != nil {
		t.Fatalf("InitCache() error = %v", err)
	}

	type args struct {
		ctx  context.Context
		zone string
		recs []libdns.Record
	}
	tests := []struct {
		name    string
		args    args
		want    []libdns.Record
		wantErr bool
	}{
		{"t1", args{
			ctx:  context.Background(),
			zone: "newzhxu.com",
			recs: []libdns.Record{libdns.RR{
				Name: "test.newzhxu.com",
				TTL:  1,
				Type: "A",
				Data: "1.1.1.1",
			}},
		}, []libdns.Record{libdns.RR{
			Name: "test.newzhxu.com",
			TTL:  1,
			Type: "A",
			Data: "1.1.1.1",
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := provider.AppendRecords(tt.args.ctx, tt.args.zone, tt.args.recs)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendRecords() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_DeleteRecords(t *testing.T) {
	provider := New()
	if err := provider.InitCache(context.Background()); err != nil {
		t.Fatalf("InitCache() error = %v", err)
	}

	type args struct {
		ctx  context.Context
		zone string
		recs []libdns.Record
	}
	tests := []struct {
		name    string
		args    args
		want    []libdns.Record
		wantErr bool
	}{
		{"t1", args{
			ctx:  context.Background(),
			zone: "newzhxu.com",
			recs: []libdns.Record{libdns.RR{
				Name: "test.newzhxu.com",
				TTL:  1,
				Type: "A",
				Data: "1.1.1.1",
			}},
		}, []libdns.Record{libdns.RR{
			Name: "test.newzhxu.com",
			TTL:  1,
			Type: "A",
			Data: "1.1.1.1",
		}}, false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := provider.DeleteRecords(tt.args.ctx, tt.args.zone, tt.args.recs)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteRecords() got = %v, want %v", got, tt.want)
			}
		})
	}
}
