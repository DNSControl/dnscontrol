var PROVIDER_NONE = NewRegistrar('none');
var PROVIDER_TRANSIP = NewDnsProvider('transip', '-');

DEFAULTS(
  DefaultTTL('1d')
)

require_glob(
  './domains/',
  false
);

require_glob(
  './domains/dnscontrol.org/',
  false
);
