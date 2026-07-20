---
name: OPENPGPKEY
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

`OPENPGPKEY` adds an [OpenPGP public key record](https://datatracker.ietf.org/doc/html/rfc7929) to the domain.

So far, no transformation is applied to the parameters. The data will be passed to the DNS server as-is. DNSControl supports both hex-encoded and base64-encoded input for the public key portion of the record.

There are multiple ways to generate the appropriately-formatted record values:

1.  By using `gpg --export-options=export-dane`:

    {% code title="Shell Transcript" %}
    ```shell-session
    $ gpg --export --export-options=export-dane example-1@dnscontrol.org
    $ORIGIN _openpgpkey.dnscontrol.org.
    ; 9305F15FF783096D39427E6D048E36367E3E3AE2
    ; Example 1 <example-1@dnscontrol.org>
    bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883d TYPE61 \# 219 (
        9833040000000116092b06010401da470f010107401471ec1d5cc4d6bbd87029
        97ed29f95f7a7bd5e179aa8d3698efc8b942eb08f5b4244578616d706c652031
        203c6578616d706c652d3140646e73636f6e74726f6c2e6f72673e887e041316
        0a00261621049305f15ff783096d39427e6d048e36367e3e3ae2050200000001
        021b01021e05021780000a0910048e36367e3e3ae2ffaa00ff4b6ad99b62da7e
        9d759abe6ae232016780c24bf5e5f869b8003be83c6a73933c0100b66ac65093
        a0fe0a434448d9996ab46412cbe7c70d5c5ab74abba4566c468d0a
    )
    ```
    {% endcode %}

    {% code title="dnsconfig.js" %}
    ```javascript
    D("dnscontrol.org", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
      OPENPGPKEY(
        "bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883d._openpgpkey",
        "9833040000000116092b06010401da470f010107401471ec1d5cc4d6bbd87029" +
        "97ed29f95f7a7bd5e179aa8d3698efc8b942eb08f5b4244578616d706c652031" +
        "203c6578616d706c652d3140646e73636f6e74726f6c2e6f72673e887e041316" +
        "0a00261621049305f15ff783096d39427e6d048e36367e3e3ae2050200000001" +
        "021b01021e05021780000a0910048e36367e3e3ae2ffaa00ff4b6ad99b62da7e" +
        "9d759abe6ae232016780c24bf5e5f869b8003be83c6a73933c0100b66ac65093" +
        "a0fe0a434448d9996ab46412cbe7c70d5c5ab74abba4566c468d0a",
      ),
    );
    ```
    {% endcode %}

2.  By using `gpg --armor` and `sha256sum`:

    {% code title="Shell Transcript" %}
    ```shell-session
    $ gpg --armor --export example-1@dnscontrol.org
    -----BEGIN PGP PUBLIC KEY BLOCK-----

    mDMEAAAAARYJKwYBBAHaRw8BAQdAFHHsHVzE1rvYcCmX7Sn5X3p71eF5qo02mO/I
    uULrCPW0JEV4YW1wbGUgMSA8ZXhhbXBsZS0xQGRuc2NvbnRyb2wub3JnPoh+BBMW
    CgAmFiEEkwXxX/eDCW05Qn5tBI42Nn4+OuIFAgAAAAECGwECHgUCF4AACgkQBI42
    Nn4+OuL/qgD/S2rZm2Lafp11mr5q4jIBZ4DCS/Xl+Gm4ADvoPGpzkzwBALZqxlCT
    oP4KQ0RI2ZlqtGQSy+fHDVxat0q7pFZsRo0K
    =JavT
    -----END PGP PUBLIC KEY BLOCK-----

    echo $(printf 'example-1' | sha256sum | head --bytes=66)
    bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883dd429ba83
    ```
    {% endcode %}

    {% code title="dnsconfig.js" %}
    ```javascript
    D("dnscontrol.org", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
      OPENPGPKEY(
        "bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883d._openpgpkey",
        "mDMEAAAAARYJKwYBBAHaRw8BAQdAFHHsHVzE1rvYcCmX7Sn5X3p71eF5qo02mO/I" +
        "uULrCPW0JEV4YW1wbGUgMSA8ZXhhbXBsZS0xQGRuc2NvbnRyb2wub3JnPoh+BBMW" +
        "CgAmFiEEkwXxX/eDCW05Qn5tBI42Nn4+OuIFAgAAAAECGwECHgUCF4AACgkQBI42" +
        "Nn4+OuL/qgD/S2rZm2Lafp11mr5q4jIBZ4DCS/Xl+Gm4ADvoPGpzkzwBALZqxlCT" +
        "oP4KQ0RI2ZlqtGQSy+fHDVxat0q7pFZsRo0K",
      ),
    );
    ```
    {% endcode %}

3.  By using the [`hash-slinger` package](https://github.com/letoams/hash-slinger/) (which is available in most Linux distro package repositories):

    {% code title="Shell Transcript" %}
    ```shell-session
    $ openpgpkey --create example-1@dnscontrol.org  # --output=rfc is the default and returns a base64-encoded key
    ; keyid: 048E36367E3E3AE2
    bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883d._openpgpkey.dnscontrol.org. IN OPENPGPKEY mDMEAAAAARYJKwYBBAHaRw8BAQdAFHHsHVzE1rvYcCmX7Sn5X3p71eF5qo02mO/IuULrCPW0JEV4YW1wbGUgMSA8ZXhhbXBsZS0xQGRuc2NvbnRyb2wub3JnPoh+BBMWCgAmFiEEkwXxX/eDCW05Qn5tBI42Nn4+OuIFAgAAAAECGwECHgUCF4AACgkQBI42Nn4+OuL/qgD/S2rZm2Lafp11mr5q4jIBZ4DCS/Xl+Gm4ADvoPGpzkzwBALZqxlCToP4KQ0RI2ZlqtGQSy+fHDVxat0q7pFZsRo0K

    $ openpgpkey --create --output=generic example-2@dnscontrol.org  # --output=generic returns a hex-encoded key
    ; keyid: 4CDE32253EDE7C0B
    6c9b19cb967b563d9d96b341ad4a89a74444c6f18e9530f4623817fe._openpgpkey.dnscontrol.org. IN TYPE61 \# 219 9833040000000116092b06010401da470f01010740416889e205cfeb00b6c10ee1ee875c2e9654fa6403dab1e2aad4e08fed5eea8bb4244578616d706c652032203c6578616d706c652d3240646e73636f6e74726f6c2e6f72673e887e0413160a0026162104fedb8dd6d3ff8e92a4a3f12f4cde32253ede7c0b050200000001021b01021e05021780000a09104cde32253ede7c0bb13d00ff68486b25a097f450f52248c0ffc5262b49e8923b49372e3a22ddc8593193e0440100923a82879140126abbf5271e68efd0e7ea050402b7cedff735ea6712e388840d
    ```
    {% endcode %}

    {% code title="dnsconfig.js" %}
    ```javascript
    D("dnscontrol.org", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
      OPENPGPKEY(
        "bb7d0cf1ee44aca0bcc0f739b77b935f13aec2fd537f5c29dedd883d._openpgpkey",
        "mDMEAAAAARYJKwYBBAHaRw8BAQdAFHHsHVzE1rvYcCmX7Sn5X3p71eF5qo02mO/IuULrCPW0JEV4YW1wbGUgMSA8ZXhhbXBsZS0xQGRuc2NvbnRyb2wub3JnPoh+BBMWCgAmFiEEkwXxX/eDCW05Qn5tBI42Nn4+OuIFAgAAAAECGwECHgUCF4AACgkQBI42Nn4+OuL/qgD/S2rZm2Lafp11mr5q4jIBZ4DCS/Xl+Gm4ADvoPGpzkzwBALZqxlCToP4KQ0RI2ZlqtGQSy+fHDVxat0q7pFZsRo0K",
      ),
      OPENPGPKEY(
        "6c9b19cb967b563d9d96b341ad4a89a74444c6f18e9530f4623817fe._openpgpkey",
        "9833040000000116092b06010401da470f01010740416889e205cfeb00b6c10ee1ee875c2e9654fa6403dab1e2aad4e08fed5eea8bb4244578616d706c652032203c6578616d706c652d3240646e73636f6e74726f6c2e6f72673e887e0413160a0026162104fedb8dd6d3ff8e92a4a3f12f4cde32253ede7c0b050200000001021b01021e05021780000a09104cde32253ede7c0bb13d00ff68486b25a097f450f52248c0ffc5262b49e8923b49372e3a22ddc8593193e0440100923a82879140126abbf5271e68efd0e7ea050402b7cedff735ea6712e388840d",
      ),
    );
    {% endcode %}
