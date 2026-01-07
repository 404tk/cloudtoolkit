# CloudToolKit

Interactive multi-cloud security assessment framework.

## Features

- **Multi-Cloud Support** - Alibaba, Tencent, Huawei, AWS, Azure, GCP, Volcengine, JDCloud
- **Asset Enumeration** - Hosts, databases, storage buckets, domains, IAM users
- **Security Testing** - Backdoor user creation, command execution, bucket dumping
- **Interactive CLI** - Tab completion, session management, credential caching

## Quick Start

```bash
# Download from releases or build from source
go build --ldflags "-s -w" -trimpath -o ctk cmd/main.go

# Run interactive console
./ctk
```

## Supported Capabilities

| Provider | Enumeration | Security Testing |
|:--------:|:-----------:|:----------------:|
| Alibaba Cloud | ECS, OSS, RAM, RDS, DNS, SLS, SMS | backdoor-user, bucket-dump, exec-command, event-dump, database-account |
| Tencent Cloud | CVM, Lighthouse, COS, CAM, CDB, DNSPod | backdoor-user, exec-command |
| Huawei Cloud | ECS, OBS, IAM, RDS | backdoor-user |
| AWS | EC2, S3, IAM | backdoor-user, bucket-dump |
| Azure | Virtual Machines, Blob Storage | - |
| GCP | Compute Engine, Cloud DNS, IAM | - |
| Volcengine | ECS, IAM | - |
| JDCloud | VM, IAM, OSS | - |

## Documentation

See [Wiki](https://github.com/404tk/cloudtoolkit/wiki) for detailed usage.

## Acknowledgements

- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
- [rapid7/metasploit-framework](https://github.com/rapid7/metasploit-framework)