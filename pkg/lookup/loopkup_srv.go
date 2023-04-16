package lookup

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

////////////////////////////////////////////////////////////////////////////////////

// LookupSRVSystemNet resolves a DNS SRV name using the Golang resolver, returning net.SRV records.
func LookupSRVSystemNet(name string, recurse bool) ([]*net.SRV, error) {
	_, records, err := net.LookupSRV("", "", name)
	if err != nil {
		return records, err
	}

	if recurse {
		for _, record := range records {
			ipname, err := net.LookupHost(record.Target)
			if err != nil {
				return records, err
			} else if len(ipname) == 0 {
				continue
			}
			record.Target = ipname[0]
		}
	}
	return records, nil
}

/////////////////////////////////////////////////////////////////////////////////////

// LookupSRVSystem resolves a DNS SRV name using the Golang resolver, returning dns.SRV records.
func LookupSRVSystem(name string, recurse bool) ([]*dns.SRV, error) {
	var dnsRecords []*dns.SRV
	netRecords, err := LookupSRVSystemNet(name, recurse)
	for _, netRecord := range netRecords {
		dnsRecords = append(dnsRecords, &dns.SRV{
			Hdr:      dns.RR_Header{},
			Priority: netRecord.Priority,
			Weight:   netRecord.Weight,
			Port:     netRecord.Port,
			Target:   netRecord.Target,
		})
	}
	return dnsRecords, err
}

/////////////////////////////////////////////////////////////////////////////////////

// LookupSRVCustom resolves a DNS name using a custom resolver (via github.com/miekg/dns), returning dns.SRV records.
func LookupSRVCustom(name string, dnsResolver string, recurse bool) ([]*dns.SRV, error) {
	c := dns.Client{}
	m := dns.Msg{}
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}
	m.SetQuestion(name, dns.TypeSRV)
	r, _, err := c.Exchange(&m, dnsResolver)
	if err != nil {
		return nil, err
	}
	if len(r.Answer) == 0 {
		return nil, nil
	}
	var records []*dns.SRV
	for _, ans := range r.Answer {
		if srvRecord, ok := ans.(*dns.SRV); ok && srvRecord != nil {
			if recurse && net.ParseIP(srvRecord.Target) == nil {
				m2 := dns.Msg{}
				m2.SetQuestion(srvRecord.Target, dns.TypeA)
				m2.RecursionDesired = true
				r2, _, err := c.Exchange(&m2, dnsResolver)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					return nil, err
				}
				if len(r2.Answer) != 0 {
					aRecord := r2.Answer[0].(*dns.A)
					srvRecord.Target = aRecord.A.String()
				}
			}
			records = append(records, srvRecord)
		}
	}
	return records, nil
}
