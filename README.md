# cloudtoolkit
Cloud Penetration Testing Toolkit

## Usage  
Reference [Wiki](https://github.com/404tk/cloudtoolkit/wiki)  

## Capability overview

|          Provider          |                   Payload                   |                          Supported                           |
| :-------------------------: | :-----------------------------------------: | :----------------------------------------------------------: |
|        Alibaba Cloud        | cloudlist<br/>backdoor-user<br/>bucket-dump<br/>event-dump<br/>exec-command<br/>database-account | ECS <br/>OSS<br/>RAM <br/>RDS <br/>SMS <br/>AliDNS<br/>SLS |
|        Tencent Cloud        |         cloudlist<br/>backdoor-user<br/>exec-command         | CVM <br/>Lighthouse<br/>COS<br/>CAM <br/>CDB <br/>DNSPod |
|        Huawei Cloud         |         cloudlist<br/>backdoor-user         | ECS <br/>OBS <br/>IAM <br/>RDS |
|       Microsoft Azure       |                  cloudlist                  |              Virtual Machines<br/>Blob Storage               |
|  AWS  | cloudlist<br/>backdoor-user<br/>bucket-dump | EC2<br/>S3 <br/>IAM |
| GCP |                  cloudlist                  |                 Compute Engine<br/>Cloud DNS<br/>IAM                 |
|         Volcengine          |                          cloudlist                           |                         ECS<br/>IAM                          |
| JDCloud | cloudlist | VM<br/>IAM<br/>OSS |

## Thanks
- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
- [rapid7/metasploit-framework](https://github.com/rapid7/metasploit-framework)