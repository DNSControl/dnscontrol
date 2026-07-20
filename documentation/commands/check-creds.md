# check-creds

This is a stand-alone utility to help verify entries in `creds.json`.

The command does a trivial operation to verify credentials.  If
successful, a list of zones will be output (which may be an empty list). If the credentials or other problems prevent this operation from executing, the exit code will be non-zero and hopefully verbose error messages will be output.

```text
Syntax:

   dnscontrol check-creds [command options] credkey

   --creds value   Provider credentials JSON file (default: "creds.json")
   --out value     Instead of stdout, write to this file

ARGUMENTS:
   credkey:  The name used in creds.json
```

The provider type is read from the `TYPE` field in `creds.json`.

## Examples

```shell
dnscontrol check-creds my_route53
```

This command is the same as `get-zones` with `--format=nameonly`

# Developer Note

This command is not implemented for all providers.

To add this to a provider, implement the get-zones subcommand.
