# Ordering of DNS records

DNSControl tries to automatically reorder the pending changes based on the dependencies of the records using [Topological sorting](https://en.wikipedia.org/wiki/Topological_sorting).
For example, if an A record and a CNAME that points to the A record are created at the same time, some providers require the A record to be created before the CNAME.

Some providers explicitly require the targets of certain records like CNAMEs to exist, and source records to be valid. This makes it not always possible to "just" apply the pending changes in any order. This is why reordering the records based on the dependencies and the type of operation is required.

## Practical example

```javascript
D("example.com", REG_NONE, DnsProvider(DNS_BIND),
    CNAME("foo", "bar"),
    A("bar", "1.2.3.4"),
);
```

`foo` requires `bar` to exist. Thus `bar` needs to exist before `foo`. But when deleting these records, `foo` needs to be deleted before `bar`.

## Unresolved records

DNSControl can produce a warning stating it found `unresolved records` this is most likely because of a cycle in the targets of your records. For instance in the code sample below both `foo` and `bar` depend on each other and thus will produce the warning.

Such updates will be done after all other updates to that domain.

In this (contrived) example, it is impossible to know which CNAME should be created first. Therefore they will be done in a non-deterministic order after all other updates to that domain:

```javascript
D("example.com", REG_NONE, DnsProvider(DNS_BIND),
    CNAME("foo", "bar"),
    CNAME("bar", "foo"),
);
```

## Disabling ordering

The re-ordering feature can be disabled using the `--disableordering` global flag (it goes before `preview` or `push`). While the code has been extensively tested, it is new and you may still find a bug.  This flag leaves the updates unordered and may require multiple `push` runs to complete the update.

If you encounter any issues with the reordering please [open an issue](https://github.com/DNSControl/dnscontrol/issues).

## Internals

DNSControl sorts all changes based on the dependencies within these changes. Each record define it's dependencies in `models.Record`. For DNSControl it doesn't matter of a CNAME's target is an A or AAAA or TXT, it will ensure all changes on the target get sorted before the depending CNAME record. The creation of the graph happens in `dnsgraph.CreateGraph([]Graphable)` and thereafter the sorting happens in `graphsort.SortUsingGraph([]Graphable)`.
The Graphable is an interface to make the sorting module more separate from the rest of the DNSControl code, currently the only Graphable implementation is `diff2.Change`.

In order to add a new sortable rtype one should add it to the `models.Record.GetGetDependencies()` and return the dependent records, this is used inside the `diff2.Change` to detect if a dependency is backwards (dependent on the old state) or forward (dependent on new state). Now the new rtype should be sorted accordingly just like MX and CNAME records.
