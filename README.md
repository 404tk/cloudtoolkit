# CloudToolKit

CloudToolKit is an adversary simulation and validation toolkit for assessing the effectiveness of CSPM, CNAPP, and related cloud detection and investigation platforms in authorized environments.

## Overview

CloudToolKit helps defenders reproduce realistic cloud security scenarios in owned labs, approved internal subscriptions, and explicitly authorized customer environments. It is designed for defensive validation through cloud asset inventory, identity and privilege abuse checks, suspicious resource activity review, and authorized instance command checks that generate realistic telemetry for detection and investigation testing.

## Features

- **Multi-Cloud Coverage** - Alibaba, Tencent, Huawei, AWS, Azure, GCP, Volcengine, JDCloud, and UCloud
- **Cloud Asset Inventory** - Hosts, databases, storage buckets, domains, IAM users, and related cloud resources
- **Defender-Side Validation Payloads** - `iam-user-check`, `bucket-check`, `instance-cmd-check`, `event-check`, `rds-account-check`, `role-binding-check`, `bucket-acl-check`, and `sa-key-check`
- **Interactive CLI** - Tab completion, session management, and credential caching
- **Lightweight Provider Clients** - AWS, Azure, Tencent, Huawei, and Alibaba integrations are being gradually decoupled from heavy official SDK paths

## Supported Capabilities

| Provider | Inventory Coverage | Validation Payloads |
|:--------:|:-----------:|:--------------------:|
| Alibaba Cloud | ECS, OSS, RAM, RDS, DNS, SLS, SMS | iam-user-check, bucket-check, instance-cmd-check, event-check, rds-account-check |
| Tencent Cloud | CVM, Lighthouse, COS, CAM, CDB, DNSPod | iam-user-check, bucket-check, instance-cmd-check |
| Huawei Cloud | ECS, OBS, IAM, RDS | iam-user-check, bucket-check |
| AWS | EC2, S3, IAM | iam-user-check, bucket-check |
| Azure | Virtual Machines, Blob Storage | role-binding-check, bucket-acl-check |
| GCP | Compute Engine, Cloud DNS, IAM | role-binding-check, sa-key-check |
| Volcengine | ECS, IAM, TOS, RDS, DNS | iam-user-check, bucket-check, instance-cmd-check |
| JDCloud | VM, LAVM, IAM, OSS | iam-user-check, bucket-check, instance-cmd-check |
| UCloud | UHost, IAM, US3, UDB, UDNS | iam-user-check |

## Example Validation Workflows

- Use `cloudlist` in an authorized environment to verify whether a CSPM or CNAPP accurately discovers compute, storage, identity, database, and DNS resources.
- Use `iam-user-check` to create or remove a test IAM user and validate identity telemetry, alerting, and persistence detection coverage.
- Use `instance-cmd-check` to generate telemetry for command execution, process correlation, and investigation workflows on a test instance.
- Use `event-check` to review cloud security events and suspicious resource operations for investigation context, enrichment quality, and timeline reconstruction.
- Use `rds-account-check` to provision read-only RDS access in an authorized environment to validate database visibility, control coverage, and investigation readiness.
- Use `role-binding-check` to bind or unbind a test principal at an authorized scope to validate role-assignment telemetry, alerting, and audit trail coverage (Azure RBAC, GCP project IAM bindings).
- Use `bucket-acl-check` to toggle storage container public-access settings in an authorized environment to validate detection coverage for unintended data exposure (Azure Blob containers).
- Use `sa-key-check` to mint or revoke a service-account key in an authorized environment to validate detection coverage for credential lifecycle abuse (GCP service accounts).

## Use Cases

- storage exposure checks in authorized environments
- IAM lifecycle checks for identity telemetry and alert validation
- role assignment / IAM binding checks for privilege change detection coverage
- service-account key lifecycle checks for long-lived credential abuse detection
- instance execution telemetry checks for detection and investigation workflows
- RDS account validation for database visibility and control verification
- cross-signal investigation testing across identity, compute, storage, and database activity

## Quick Start

```bash
# Download from releases or build from source
go build --ldflags "-s -w" -trimpath -o ctk cmd/main.go

# Run interactive console
./ctk
```

## Responsible Use

CloudToolKit is intended only for owned environments, lab environments, approved internal subscriptions, and explicitly authorized customer environments. It is designed to help defenders verify detection coverage, telemetry quality, investigation workflows, and control effectiveness. It is not intended for unauthorized access, third-party abuse, or covert real-world intrusion activity.

## Non-Goals

CloudToolKit is not positioned as:

- an unauthorized offensive toolkit
- a stealth or bypass framework
- a weaponized intrusion utility
- guidance for abuse against third-party environments

## Documentation

See [Wiki](https://github.com/404tk/cloudtoolkit/wiki) for detailed usage.

## Acknowledgements

- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
