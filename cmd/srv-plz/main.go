package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/neomantra/srv-plz/pkg/lookup"

	"github.com/miekg/dns"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormatShort string = `usage:  %s <options> [service1 [service2 [...]]] [-- command]`

var usageFormat string = `usage:  %s <options> [service1 [service2 [...]]] [-- command]

srv-plz resolves DNS SRV records and outputs the result.

The resolver is specified with "--dns <IP:port>" argument or by setting
the SRV_DNS environment variable.  If only an IP address is set, port 53 is used.

If no DNS resolver is specified, the system resolver is used.

The default output is "host:port".  This may be customized with the --template
argument.  Possible fields are Target, Port, Priority, and Weight.
The default template is "{{.Target}}:{{.Port}}\n" for SRV records and "{{.Target}}\n" for A/AAAA records.

If "--command" is flagged, each SRV record will be injected into the command
specified after "--", using "%%SRV%%" or the "--match" argument as a matcher.  Example:

    srv-plz webserver.service.consul -r -c -- curl https://%%SRV%%/health

Arguments:
`

const DEFAULT_SRV_TEMPLATE_STR = "{{.Target}}:{{.Port}}"
const DEFAULT_A_TEMPLATE_STR = "{{.Target}}"

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var dnsServer string
	var recurse bool
	var numLimit uint32
	var templateStr string
	var invokeCommand bool
	var srvMatcher string
	var checkARecord bool
	var checkAAAARecord bool
	var showHelp bool

	pflag.StringVarP(&dnsServer, "dns", "d", "", "DNS resolver to use. Must be in form IP (using port 53) or IP:port")
	pflag.BoolVarP(&recurse, "recurse", "r", false, "recurse with the same resolver")
	pflag.Uint32VarP(&numLimit, "limit", "l", 1, "only return N records")
	pflag.StringVarP(&templateStr, "template", "t", DEFAULT_SRV_TEMPLATE_STR, "output using template")
	pflag.BoolVarP(&invokeCommand, "command", "c", false, "for each record, invoke exec.Command on the args after '--', replacing %SRV% with its template")
	pflag.StringVarP(&srvMatcher, "match", "m", "%SRV%", "specify forward args after '--' to shell with <srv> replaced by the lookup")
	pflag.BoolVarP(&checkARecord, "a", "a", false, "Check A records, not SRV records")
	pflag.BoolVarP(&checkAAAARecord, "aaaa", "6", false, "Check AAAA records, not SRV records")
	pflag.BoolVarP(&showHelp, "help", "h", false, "show help")
	pflag.Parse()

	var serviceArgs, commandArgs []string
	if pflag.CommandLine.ArgsLenAtDash() == -1 {
		serviceArgs = pflag.Args()
		commandArgs = nil
	} else {
		serviceArgs = pflag.Args()[0:pflag.CommandLine.ArgsLenAtDash()]
		commandArgs = pflag.Args()[pflag.CommandLine.ArgsLenAtDash():]
	}
	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	if (checkARecord || checkAAAARecord) && templateStr == DEFAULT_SRV_TEMPLATE_STR {
		// switch default template if we are checking A/AAAA records
		// but only if user has not changed it
		templateStr = DEFAULT_A_TEMPLATE_STR
	}

	// setup resolver
	if len(dnsServer) == 0 {
		// try from environment if not already set by CLI
		dnsServer = os.Getenv("SRV_DNS")
	}
	if len(dnsServer) != 0 {
		// check addr:port form is valid
		_, _, err := net.SplitHostPort(dnsServer)
		if err != nil {
			// https://cs.opensource.google/go/go/+/refs/tags/go1.20.3:src/net/ipsock.go;l=166
			const missingPort = "missing port in address"
			if strings.Contains(err.Error(), missingPort) { // this is OK, we default to port 53
				dnsServer += ":53"
			} else {
				fmt.Fprintf(os.Stderr, "invalid --dns error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// setup output template
	tmpl, err := template.New("srv").Parse(templateStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --template error: %v\n", err)
		os.Exit(1)
	}

	// lookup the services
	if len(serviceArgs) == 0 {
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		fmt.Fprintf(os.Stderr, "\ntry     %s --help\n", os.Args[0])
		os.Exit(0)
	}

	for _, serviceName := range serviceArgs {
		var srvRecords []*dns.SRV
		// check A/AAAA records?
		if checkARecord || checkAAAARecord {
			// lookup A/AAAA using system resolver?
			if len(dnsServer) == 0 {
				records, err := lookup.LookupSystem(serviceName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					continue
				}
				for _, record := range records {
					// filter out A/AAAA records, per flags
					if (checkARecord && checkAAAARecord) ||
						((checkARecord && strings.Contains(record, ".")) ||
							(checkAAAARecord && strings.Contains(record, ":"))) {
						srvRecords = append(srvRecords, &dns.SRV{Target: record})
					}
				}
			} else { // lookup A/AAAA using custom resolver
				// lookup A using custom resolver?
				if checkARecord {
					records, err := lookup.LookupACustom(serviceName, dnsServer)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						continue
					}
					for _, record := range records {
						srvRecords = append(srvRecords, &dns.SRV{Target: record})
					}
				}
				// lookup AAAA using custom resolver?
				if checkAAAARecord {
					records, err := lookup.LookupAAAACustom(serviceName, dnsServer)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						continue
					}
					for _, record := range records {
						srvRecords = append(srvRecords, &dns.SRV{Target: record})
					}
				}
			}
		} else { // Lookup SRV record
			if len(dnsServer) == 0 { // system resolver?
				srvRecords, err = lookup.LookupSRVSystem(serviceName, recurse)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					continue
				}
			} else { // custom resolver
				srvRecords, err = lookup.LookupSRVCustom(serviceName, dnsServer, recurse)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					continue
				}
			}
		}
		handleRecords(srvRecords, invokeCommand, srvMatcher, commandArgs, *tmpl)
	}
}

func handleRecords(records []*dns.SRV, invokeCommand bool, srvMatcher string, commandArgs []string, tmpl template.Template) {
	for _, record := range records {
		builder := new(strings.Builder)
		err := tmpl.Execute(builder, record)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template failed: %v\n", err)
			continue
		}
		// check to invoke Command
		if invokeCommand {
			err := replaceAndCommand(builder.String(), srvMatcher, commandArgs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		} else {
			// otherwise write to stdout
			os.Stdout.WriteString(builder.String())
			os.Stdout.WriteString("\n")
		}
	}
}

func replaceAndCommand(record string, invokeMatcher string, commandArgs []string) error {
	xargs := []string{}
	for _, xarg := range commandArgs {
		xargs = append(xargs, strings.ReplaceAll(xarg, invokeMatcher, record))
	}
	if len(xargs) == 0 {
		return nil
	}
	return invokeCommand(xargs[0], xargs[1:]...)
}

func invokeCommand(command string, args ...string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	os.Stdout.Write(stdout.Bytes())
	os.Stderr.Write(stderr.Bytes())
	return err
}
