package cflibdns

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func Test_cache_addZone(t *testing.T) {
	provider := New(logrus.New())

	type args struct {
		z []*zone
	}
	tests := []struct {
		name string
		args args
	}{
		{"t1", args{z: []*zone{{id: "1", name: "example.Com."}}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			provider.cache.setZones(tt.args.z)

		})
	}
}
