package lookup

import (
	"net"
	"strings"

	"github.com/miekg/dns"
)

////////////////////////////////////////////////////////////////////////////////////

// LookupSystem resolves a DNS name using the Golang resolver, returning both A and AAAA records.
func LookupSystem(name string) ([]string, error) {
	records, err := net.LookupHost(name)
	if err != nil {
		return records, err
	}
	return records, nil
}

/////////////////////////////////////////////////////////////////////////////////////

// LookupACustom resolves a DNS name using a custom resolver (via github.com/miekg/dns), returning A records.
func LookupACustom(name string, dnsResolver string) ([]string, error) {
	c := dns.Client{}
	m := dns.Msg{}
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}
	m.SetQuestion(name, dns.TypeA)
	r, _, err := c.Exchange(&m, dnsResolver)
	if err != nil {
		return nil, err
	}
	if len(r.Answer) == 0 {
		return nil, nil
	}
	var records []string
	for _, ans := range r.Answer {
		if aRecord, ok := ans.(*dns.A); ok && aRecord != nil {
			records = append(records, aRecord.A.String())
		}
	}
	return records, nil
}

/////////////////////////////////////////////////////////////////////////////////////

// LookupAAAACustom resolves a DNS name using a custom resolver (via github.com/miekg/dns), returning AAAA records.
func LookupAAAACustom(name string, dnsResolver string) ([]string, error) {
	c := dns.Client{}
	m := dns.Msg{}
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}
	m.SetQuestion(name, dns.TypeAAAA)
	r, _, err := c.Exchange(&m, dnsResolver)
	if err != nil {
		return nil, err
	}
	if len(r.Answer) == 0 {
		return nil, nil
	}
	var records []string
	for _, ans := range r.Answer {
		if aaaaRecord, ok := ans.(*dns.AAAA); ok && aaaaRecord != nil {
			records = append(records, aaaaRecord.AAAA.String())
		}
	}
	return records, nil
}
