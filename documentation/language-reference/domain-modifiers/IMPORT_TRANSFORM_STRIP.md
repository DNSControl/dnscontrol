---
name: IMPORT_TRANSFORM_STRIP
parameters:
  - transform table
  - domain
  - ttl
  - suffixstrip
  - modifiers...
ts_ignore: true
---

{% hint style="warning" %}
Don't use this feature. It was added for a very specific situation at Stack Overflow.
{% endhint %}

`IMPORT_TRANSFORM_STRIP` is the same as `IMPORT_TRANSFORM` with an additional parameter: `suffixstrip`.

When `IMPORT_TRANSFORM_STRIP` is generating the label for new records, it checks the label.  If the label ends with `.` + `suffixstrip`, that suffix is removed. If the label does not end with `suffixstrip`, an error is returned.

For CNAMEs, the `suffixstrip` is stripped from the beginning (prefix) of the target domain.

For example, if the domain is `com.extra` and the label is `foo.com`, `IMPORT_TRANSFORM` would generate a label `foo.com.com.extra`. `IMPORT_TRANSFORM_STRIP(... , 'com')` would generate the label `foo.com.extra` instead.

In the case of a CNAME, if the target is `foo.com.`, the new target would be `foo.com.extra`.
