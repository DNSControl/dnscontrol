interface Domain {
    name: string;
    subdomain: string;
    registrar: unknown;
    meta: Record<string, unknown>;
    records: DNSRecord[];
    dnsProviders: Record<string, unknown>;
    dnsProviderMetadata?: Record<string, ProviderConfigMetadata>;
    defaultTTL: number;
    nameservers: unknown[];
    ignored_names: unknown[];
    ignored_targets: unknown[];
    [key: string]: unknown;
}

type ProviderConfigMetadata =
    | null
    | boolean
    | number
    | string
    | ProviderConfigMetadata[]
    | { [key: string]: ProviderConfigMetadata };

interface DNSRecord {
    type: string;
    meta: Record<string, unknown>;
    ttl: number;
}

type DomainModifier =
    | ((domain: Domain) => void)
    | Partial<Domain>
    | DomainModifier[];

type RecordModifier =
    | ((record: DNSRecord) => void)
    | Partial<DNSRecord['meta']>
    | RecordModifier[];

type Duration =
    | `${number}${'s' | 'm' | 'h' | 'd' | 'w' | 'n' | 'y' | ''}`
    | number /* seconds */;
