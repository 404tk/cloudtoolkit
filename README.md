# CloudToolKit

<strong>English</strong> | <a href="./docs/README.zh-CN.md">简体中文</a>

> Multi-cloud defensive validation toolkit for CSPM / CNAPP detection, telemetry, and investigation workflows in authorized environments.

CloudToolKit gives security teams a practical way to verify whether cloud controls are discoverable, detectable, alertable, and investigable before those gaps matter in production.

## Why CloudToolKit

| Advantage | What it gives defenders |
|---|---|
| 9-cloud coverage | One workflow across major global and China cloud providers. |
| Asset-first inventory | Hosts, databases, buckets, domains, accounts, logs, SMS assets, and billing-plane signals where supported. |
| Validation payloads | Focused checks for identity lifecycle, credential lifecycle, role bindings, storage exposure, audit events, instance command telemetry, and database account changes. |
| Replay mode | `demo` drives providers against in-memory replay fixtures, so detection logic can be tested without live cloud calls. |
| Conservative claims | Capabilities are advertised only when drivers, replay paths, and focused tests are in place. |

## Capability Matrix

Every provider supports `cloudlist` asset enumeration. Asset categories include host / database / bucket / domain / account / log / sms / balance where the cloud has a native equivalent.

Validation payload coverage:

<table>
  <tr>
    <th align="left" width="170">Cloud</th>
    <th align="center">iam</th>
    <th align="center">bucket</th>
    <th align="center">event</th>
    <th align="center">cmd</th>
    <th align="center">rds</th>
    <th align="center">role</th>
    <th align="center">acl</th>
    <th align="center">cred</th>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/aws.svg" width="28" height="28" alt="AWS icon">&nbsp;<strong>AWS</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/azure.svg" width="28" height="28" alt="Azure icon">&nbsp;<strong>Azure</strong></td>
    <td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/gcp.svg" width="28" height="28" alt="GCP icon">&nbsp;<strong>GCP</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/alibaba.svg" width="28" height="28" alt="Alibaba icon">&nbsp;<strong>Alibaba</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/tencent.svg" width="28" height="28" alt="Tencent icon">&nbsp;<strong>Tencent</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/huawei.svg" width="28" height="28" alt="Huawei icon">&nbsp;<strong>Huawei</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/volcengine.svg" width="28" height="28" alt="Volcengine icon">&nbsp;<strong>Volcengine</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/jdcloud.svg" width="28" height="28" alt="JDCloud icon">&nbsp;<strong>JDCloud</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="docs/icons/ucloud.svg" width="28" height="28" alt="UCloud icon">&nbsp;<strong>UCloud</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td>
  </tr>
</table>


Legend: `iam` = user lifecycle · `bucket` = object visibility · `event` = audit log review · `cmd` = instance command telemetry · `rds` = database account lifecycle · `role` = privilege binding change · `acl` = storage exposure · `cred` = long-lived credential lifecycle. `—` = no native equivalent or pending validation.

## Quick Start

```bash
go build --ldflags "-s -w" -trimpath -o ctk cmd/main.go
./ctk                                    # interactive REPL
./ctk <provider> <action> [args] [flags] # headless one-shot
```

Try `demo` inside the REPL to drive any provider against an in-memory replay (no live cloud calls).

## Responsible Use

Use only on owned, lab, internal, or explicitly authorized customer environments to verify detection coverage, telemetry quality, investigation workflow, and control effectiveness. CloudToolKit is not a stealth, bypass, or unauthorized intrusion utility and must not be used against third-party environments without permission.

## Documentation

- [Wiki](https://github.com/404tk/cloudtoolkit/wiki) — usage, payload references, replay walkthroughs

## Acknowledgements

- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
