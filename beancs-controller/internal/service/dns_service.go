package service

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v2"
	"github.com/cloudflare/cloudflare-go/v2/dns"
	"github.com/cloudflare/cloudflare-go/v2/option"
	"github.com/zeturn/beancs-controller/internal/model"
)

type DNSService struct {
	IngressIP string
}

func NewDNSService(ingressIP string) *DNSService {
	return &DNSService{IngressIP: ingressIP}
}

func (s *DNSService) CreateRecord(ctx context.Context, token string, cred model.CloudflareCredential, project *model.Project) (*model.DNSRecord, error) {
	fqdn := project.Domain
	if fqdn == "" {
		fqdn = fmt.Sprintf("%s.%s", project.Subdomain, cred.Domain)
	}
	return s.CreateRecordForHost(ctx, token, cred, project.Name, fqdn)
}

func (s *DNSService) CreateRecordForHost(ctx context.Context, token string, cred model.CloudflareCredential, projectName, fqdn string) (*model.DNSRecord, error) {
	client := cloudflare.NewClient(option.WithAPIToken(token))
	result, err := client.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.F(cred.ZoneID),
		Record: dns.ARecordParam{
			Type:    cloudflare.F(dns.ARecordTypeA),
			Name:    cloudflare.F(fqdn),
			Content: cloudflare.F(s.IngressIP),
			Proxied: cloudflare.F(true),
			TTL:     cloudflare.F(dns.TTL1),
			Comment: cloudflare.F("BeanCS managed - project: " + projectName),
		},
	})
	if err != nil {
		return nil, err
	}
	return &model.DNSRecord{
		CloudflareCredentialID: cred.ID,
		CloudflareRecordID:     result.ID,
		Name:                   result.Name,
		Type:                   string(result.Type),
		Content:                fmt.Sprint(result.Content),
		Proxied:                result.Proxied,
	}, nil
}

func (s *DNSService) DeleteRecord(ctx context.Context, token, zoneID, recordID string) error {
	client := cloudflare.NewClient(option.WithAPIToken(token))
	_, err := client.DNS.Records.Delete(ctx, recordID, dns.RecordDeleteParams{ZoneID: cloudflare.F(zoneID)})
	return err
}
