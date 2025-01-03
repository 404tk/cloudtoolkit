# cloudtoolkit
Cloud Penetration Testing Toolkit

## Usage  
Reference [Wiki](https://github.com/404tk/cloudtoolkit/wiki)  

## Capability overview

|          Provider          |                   Payload                   |                          Supported                           |
| :-------------------------: | :-----------------------------------------: | :----------------------------------------------------------: |
|        Alibaba Cloud        | cloudlist<br/>backdoor-user<br/>bucket-dump<br/>event-dump<br/>exec-command<br/>database-account | ECS (Elastic Compute Service)<br/>OSS (Object Storage Service)<br/>RAM (Resource Access Management)<br/>RDS (Relational Database Service)<br/>SMS (Short Message Service)<br/>AliDNS<br/>SLS (Simple Log Service) |
|        Tencent Cloud        |         cloudlist<br/>backdoor-user<br/>exec-command         | CVM (Cloud Virtual Machine)<br/>Lighthouse<br/>COS (Cloud Object Storage)<br/>CAM (Cloud Access Management)<br/>CDB (Cloud DataBase)<br/>DNSPod |
|        Huawei Cloud         |         cloudlist<br/>backdoor-user         | ECS (Elastic Cloud Server)<br/>OBS (Object Storage Service)<br/>IAM (Identity and Access Management)<br/>RDS (Relational Database Service) |
|       Microsoft Azure       |                  cloudlist                  |              Virtual Machines<br/>Blob Storage               |
|  AWS (Amazon web services)  | cloudlist<br/>backdoor-user<br/>bucket-dump | EC2 (Elastic Compute Cloud)<br/>S3 (Simple Storage Service)<br/>IAM (Identity and Access Management) |
| GCP (Google Cloud Platform) |                  cloudlist                  |                 Compute Engine<br/>Cloud DNS<br/>Identity and Access Management                 |
|         Volcengine          |                          cloudlist                           |                         ECS<br/>IAM                          |

## Thanks
- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
- [rapid7/metasploit-framework](https://github.com/rapid7/metasploit-framework)