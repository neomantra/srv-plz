package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"os"
	"os/exec"
	"strings"

	"neomantra/srv-plz/pkg/lookup"

	"github.com/miekg/dns"
	"github.com/spf13/pflag"
)

/////////////////////////////////////////////////////////////////////////////////////

var usageFormatShort string = `usage:  %s <options> [service1 [service2 [...]]] [-- command]`

var usageFormat string = `usage:  %s <options> [service1 [service2 [...]]] [-- command]

srv-plz resolves DNS SRV records and outputs the result.

The resolver is specified with "--dns <ip:port>" argument or by setting
the SRV_DNS environment variable.  The CLI argument takes precedent.

If no DNS resolver is specified, the system resolver is used.

The default output is "host:port".  This may be customized with the --template
argument.  Possible fields are Target, Port, Priority, and Weight.
Thus the default template is "{{.Target}}:{{.Port}}\n".

If "--command" is flagged, each SRV record will be injected into the command
specified after "--", using "%%SRV%%" or the "--match" argument as a matcher.  Example:

    srv-plz webserver.service.consul -r -c -- curl https://%%SRV%%/health

Arguments:
`

/////////////////////////////////////////////////////////////////////////////////////

func main() {
	var dnsServer string
	var recurse bool
	var numLimit uint32
	var templateStr string
	var invokeCommand bool
	var srvMatcher string
	var showHelp bool

	pflag.StringVarP(&dnsServer, "dns", "d", "", "DNS resolver to use (must be in form IP:port)")
	pflag.BoolVarP(&recurse, "recurse", "r", false, "recurse with the same resolver")
	pflag.Uint32VarP(&numLimit, "limit", "l", 1, "only return N records")
	pflag.StringVarP(&templateStr, "template", "t", "{{.Target}}:{{.Port}}", "output using template")
	pflag.BoolVarP(&invokeCommand, "command", "c", false, "for each record, invoke exec.Command on the args after '--', replacing %SRV% with its template")
	pflag.StringVarP(&srvMatcher, "match", "m", "%SRV%", "specify forward args after '--' to shell with <srv> replaced by the lookup")
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
	if len(serviceArgs) == 0 {
		fmt.Fprintf(os.Stderr, usageFormatShort, os.Args[0])
		fmt.Fprintf(os.Stderr, "\ntry     %s --help\n", os.Args[0])
		os.Exit(0)
	}

	for _, service := range serviceArgs {
		var records []*dns.SRV
		var err error
		if len(dnsServer) != 0 {
			records, err = lookup.LookupSRVCustom(service, dnsServer, recurse)
		} else {
			records, err = lookup.LookupSRVSystem(service, recurse)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}

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
