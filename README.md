# srv-plz - SRV record extractor

## *Service Please!*

`srv-plz` looks up a [DNS SRV record](https://en.wikipedia.org/wiki/SRV_record) from the specified DNS server
and outputs the result.


```
$ SRV_DNS=127.0.0.1:8500 srv-plz example.service.consul

```

## Usage

```
$ srv-plz --help
usage:  srv-plz <options> [service1 [service2 [...]]]

srv-plz resolves DNS SRV records and outputs the result.

The resolver is specified with "--dns <ip:port>" argument or by setting
the SRV_DNS environment variable.  The CLI argument takes precedent.

If no DNS resolver is specified, the system resolver is used.

The default output is "host:port".  This may be customized with the --template
argument.  Possible fields are Target, Port, Priority, and Weight.
Thus the default template is "{{.Target}}:{{.Port}}\n".

  -d, --dns string        DNS resolver to use (must be in form IP:port)
  -h, --help              show help
  -l, --limit uint32      only return N records (default 1)
  -r, --recurse           recurse with the same resolver
  -t, --template string   output using template (default "{{.Target}}:{{.Port}}\n")
```

----

## Installing

Binaries for multiple platforms are [released on GitHub](https://github.com/neomantra/srv-plz/releases) through [GitHub Actions](https://github.com/neomantra/srv-plz/actions).

You can also install for various platforms with [Homebrew](https://brew.sh) from [`neomantra/homebrew-tap`](https://github.com/neomantra/homebrew-tap):

```
brew tap neomantra/homebrew-tap
brew install srv-plz
```

----

## Example Usage

Lookup with system resolver, without or with recursion:

```
$ srv-plz _http._tcp.mxtoolbox.com
mxtoolbox.com.:80

$ srv-plz -r _http._tcp.mxtoolbox.com
13.225.202.38:80
```

Lookup with custom resolver, either via CLI or `SRV_DNS` environment variable:

```
$ srv-plz -d 10.4.20.69:8600 -r webserver.service.consul 
10.4.20.69:55420

$ SRV_DNS=10.4.20.69:8600 srv-plz -r webserver.service.consul 
10.4.20.69:55420
```

Lookup with a custom output template.  Note the shell expression `$'\n'` is used to include a newline at the end.

```
$ srv-plz -d 10.4.20.69:8600 -r webserver.service.consul -t $'t:{{.Target}} p:{{.Port}} w:{{.Weight}} p:{{.Priority}}\n'
t:10.0.10.25 p:26877 w:1 p:1
```

----

## Building

Building is performed with [task](https://taskfile.dev/):

```
$ task
task: [build] go build -o srv-plz cmd/srv-plz/main.go
```

----

## Credits and License

Thanks to [github.com/miekg/dns](https://github.com/miekg/dns) for the heavy lifting.

Copyright (c) 2022 Neomantra BV.  Authored by Evan Wies.

Released under the [MIT License](https://en.wikipedia.org/wiki/MIT_License), see [LICENSE.txt](./LICENSE.txt).
