package main

import (
	"fmt"
	"html/template"
	"net"
	"os"

	"neomantra/srv-plz/pkg/lookup"

	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormatShort string = `usage:  %s <options> [service1 [service2 [...]]]`

var usageFormat string = `usage:  %s <options> [service1 [service2 [...]]]

srv-plz resolves DNS SRV records and outputs the result.

The resolver is specified with "--dns <ip:port>" argument or by setting
the SRV_DNS environment variable.  The CLI argument takes precedent.

If no DNS resolver is specified, the system resolver is used.

The default output is "host:port".  This may be customized with the --template
argument.  Possible fields are Target, Port, Priority, and Weight.
Thus the default template is "{{.Target}}:{{.Port}}\n".

`

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var dnsServer string
	var recurse bool
	var numLimit uint32
	var templateStr string
	var showHelp bool

	pflag.StringVarP(&dnsServer, "dns", "d", "", "DNS resolver to use (must be in form IP:port)")
	pflag.BoolVarP(&recurse, "recurse", "r", false, "recurse with the same resolver")
	pflag.Uint32VarP(&numLimit, "limit", "l", 1, "only return N records")
	pflag.StringVarP(&templateStr, "template", "t", "{{.Target}}:{{.Port}}\n", "output using template")
	pflag.BoolVarP(&showHelp, "help", "h", false, "show help")
	pflag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stdout, usageFormat, os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
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
			fmt.Fprintf(os.Stderr, "invalid --dns error: %v\n", err)
			os.Exit(1)
		}
	}

	// setup output template
	tmpl, err := template.New("srv").Parse(templateStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --template error: %v\n", err)
		os.Exit(1)
	}

	// lookup the services
	services := pflag.Args()
	if len(services) == 0 {
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		fmt.Fprintf(os.Stderr, "\ntry     %s --help\n", os.Args[0])
		os.Exit(0)
	}
	for _, service := range services {
		if len(dnsServer) != 0 {
			records, err := lookup.LookupSRVCustom(service, dnsServer, recurse)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			for _, record := range records {
				err := tmpl.Execute(os.Stdout, record)
				if err != nil {
					fmt.Fprintf(os.Stderr, "template failed: %v\n", err)
				}
			}
		} else {
			records, err := lookup.LookupSRVSystem(service, recurse)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			for _, record := range records {
				err := tmpl.Execute(os.Stdout, record)
				if err != nil {
					fmt.Fprintf(os.Stderr, "template failed: %v\n", err)
				}
			}
		}
	}
}
