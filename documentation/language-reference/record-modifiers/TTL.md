---
name: TTL
parameters:
  - ttl
parameter_types:
  ttl: Duration
---

TTL sets the TTL for a single record only. This will take precedence
over the domain's [DefaultTTL](../domain-modifiers/DefaultTTL.md) if supplied.

The value can be:

  * An integer (number of seconds). Example: `600`
  * A string: Integer with single-letter unit: Example: `5m`
  * The unit denotes:
    * s (seconds)
    * m (minutes)
    * h (hours)
    * d (days)
    * w (weeks)
    * n (nonths) (30 days in a nonth)
    * y (years) (If you set a TTL to a year, we assume you also do crossword puzzles in pen. Show off!)
    * If no unit is specified, the default is seconds.
  * We highly recommend using units instead of the number of seconds. Would your coworkers understand your intention better if you wrote `14400` or `'4h'`?

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  DefaultTTL(2000),
  A("@","1.2.3.4"), // uses default
  A("foo", "2.3.4.5", TTL(500)), // overrides default
  A("demo1", "3.4.5.11", TTL("5d")),  // 5 days
  A("demo2", "3.4.5.12", TTL("5w")),  // 5 weeks
);
```
{% endcode %}
