package service

import (
	"fmt"
	"testing"
)

func TestDomainMatchesZone(t *testing.T) {
	cases := []struct {
		host string
		zone string
		want bool
	}{
		{host: "issuetick.beancs.com", zone: "beancs.com", want: true},
		{host: "beancs.com", zone: "beancs.com", want: true},
		{host: "issuetick.beancs.com.", zone: "beancs.com.", want: true},
		{host: "issuetick.beancs.com", zone: "hollowdata.com", want: false},
		{host: "fakebeancs.com", zone: "beancs.com", want: false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_%s", tc.host, tc.zone), func(t *testing.T) {
			if got := domainMatchesZone(tc.host, tc.zone); got != tc.want {
				t.Fatalf("domainMatchesZone(%q, %q) = %v, want %v", tc.host, tc.zone, got, tc.want)
			}
		})
	}
}

func TestCloudflareDuplicateRecordDetectionAcceptsCodeOnly(t *testing.T) {
	err := fmt.Errorf(`POST "https://api.cloudflare.com/client/v4/zones/example/dns_records": 400 Bad Request {"errors":[{"code":81058}]}`)
	if !isCloudflareDuplicateRecord(err) {
		t.Fatal("expected Cloudflare duplicate record error")
	}
}
