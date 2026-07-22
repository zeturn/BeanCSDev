package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	if existing, ok, err := s.findARecord(ctx, token, cred, fqdn); err != nil {
		return nil, err
	} else if ok {
		if existing.Content == s.IngressIP && !existing.Proxied {
			return existing, nil
		}
		return s.updateARecord(ctx, token, cred, existing, projectName)
	}
	client := cloudflare.NewClient(option.WithAPIToken(token))
	result, err := client.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.F(cred.ZoneID),
		Record: dns.ARecordParam{
			Type:    cloudflare.F(dns.ARecordTypeA),
			Name:    cloudflare.F(fqdn),
			Content: cloudflare.F(s.IngressIP),
			Proxied: cloudflare.F(false),
			TTL:     cloudflare.F(dns.TTL1),
			Comment: cloudflare.F("BeanCS managed - project: " + projectName),
		},
	})
	if err != nil {
		if isCloudflareDuplicateRecord(err) {
			if existing, ok, findErr := s.findARecord(ctx, token, cred, fqdn); findErr != nil {
				return nil, findErr
			} else if ok {
				if existing.Content == s.IngressIP && !existing.Proxied {
					return existing, nil
				}
				return s.updateARecord(ctx, token, cred, existing, projectName)
			}
		}
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

func (s *DNSService) findARecord(ctx context.Context, token string, cred model.CloudflareCredential, fqdn string) (*model.DNSRecord, bool, error) {
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s&per_page=100", url.PathEscape(cred.ZoneID), url.QueryEscape(strings.TrimSpace(fqdn)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	out, err := cloudflareDo(ctx, &http.Client{Timeout: 15 * time.Second}, req)
	if err != nil {
		return nil, false, err
	}
	results, _ := out["result"].([]any)
	var first *model.DNSRecord
	for _, item := range results {
		rec, ok := cloudflareRecordFromAPI(cred, item)
		if !ok {
			continue
		}
		if first == nil {
			first = rec
		}
		if dnsNamesEqual(rec.Name, fqdn) {
			return rec, true, nil
		}
	}
	if first != nil {
		return first, true, nil
	}
	return nil, false, nil
}

func (s *DNSService) updateARecord(ctx context.Context, token string, cred model.CloudflareCredential, record *model.DNSRecord, projectName string) (*model.DNSRecord, error) {
	body, err := json.Marshal(map[string]any{
		"type":    "A",
		"name":    record.Name,
		"content": s.IngressIP,
		"proxied": false,
		"ttl":     1,
		"comment": "BeanCS managed - project: " + projectName,
	})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", url.PathEscape(cred.ZoneID), url.PathEscape(record.CloudflareRecordID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	out, err := cloudflareDo(ctx, &http.Client{Timeout: 15 * time.Second}, req)
	if err != nil {
		return nil, err
	}
	updated, ok := cloudflareRecordFromAPI(cred, out["result"])
	if !ok {
		return nil, fmt.Errorf("cloudflare returned malformed DNS record")
	}
	return updated, nil
}

func cloudflareRecordFromAPI(cred model.CloudflareCredential, item any) (*model.DNSRecord, bool) {
	raw, ok := item.(map[string]any)
	if !ok {
		return nil, false
	}
	record := &model.DNSRecord{
		CloudflareCredentialID: cred.ID,
		CloudflareZoneID:       cred.ZoneID,
		CloudflareRecordID:     strings.TrimSpace(fmt.Sprint(raw["id"])),
		Name:                   strings.TrimSpace(fmt.Sprint(raw["name"])),
		Type:                   strings.TrimSpace(fmt.Sprint(raw["type"])),
		Content:                strings.TrimSpace(fmt.Sprint(raw["content"])),
	}
	if proxied, ok := raw["proxied"].(bool); ok {
		record.Proxied = proxied
	}
	return record, record.CloudflareRecordID != "" && record.Name != ""
}

func (s *DNSService) EnsureRecordDNSOnly(ctx context.Context, token string, cred model.CloudflareCredential, record model.DNSRecord) error {
	if record.CloudflareRecordID == "" {
		return fmt.Errorf("cloudflare record id is empty")
	}
	body, err := json.Marshal(map[string]any{"proxied": false})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", url.PathEscape(cred.ZoneID), url.PathEscape(record.CloudflareRecordID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	_, err = cloudflareDo(ctx, &http.Client{Timeout: 15 * time.Second}, req)
	return err
}

func (s *DNSService) DeleteRecord(ctx context.Context, token, zoneID, recordID string) error {
	client := cloudflare.NewClient(option.WithAPIToken(token))
	_, err := client.DNS.Records.Delete(ctx, recordID, dns.RecordDeleteParams{ZoneID: cloudflare.F(zoneID)})
	if isCloudflareRecordNotFound(err) {
		return nil
	}
	return err
}

func isCloudflareRecordNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "404 Not Found") ||
		strings.Contains(message, "Record does not exist") ||
		strings.Contains(message, `"code":81044`)
}

func isCloudflareDuplicateRecord(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "81058") ||
		strings.Contains(strings.ToLower(message), "identical record already exists")
}

func dnsNamesEqual(a, b string) bool {
	a = strings.ToLower(strings.Trim(strings.TrimSpace(a), "."))
	b = strings.ToLower(strings.Trim(strings.TrimSpace(b), "."))
	return a == b
}
