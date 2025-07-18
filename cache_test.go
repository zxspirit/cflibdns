package provider

import "testing"

func Test_cache_addZone(t *testing.T) {
	type fields struct {
		zones []*zone
	}
	type args struct {
		z *zone
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"t1", fields{zones: []*zone{}}, args{z: &zone{id: "1", name: "example.Com."}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				zones: tt.fields.zones,
			}
			c.addZone(tt.args.z)
		})
	}
}
