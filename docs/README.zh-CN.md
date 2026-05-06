# CloudToolKit

<a href="../README.md">English</a> | <strong>简体中文</strong>

> 面向授权环境的多云防御验证工具包，用于验证 CSPM / CNAPP 检测、遥测和调查流程。

CloudToolKit 帮助安全团队在真实差距影响生产环境之前，验证云上控制面和数据面信号是否可发现、可检测、可告警、可关联、可调查。

## 为什么使用 CloudToolKit

| 优势 | 给防御团队带来的价值 |
|---|---|
| 9 家云覆盖 | 用一套工作流覆盖主流国际云与国内云环境。 |
| 资产优先 | 盘点主机、数据库、存储桶、域名、账号、日志、短信资产和账单侧信号。 |
| 验证载荷 | 聚焦身份生命周期、长期凭证、角色绑定、存储暴露、审计事件、实例命令遥测和数据库账号变更。 |
| Replay 模式 | REPL 中的 `demo` 使用内存 replay fixture，无需真实云调用即可验证检测逻辑。 |
| 保守声明 | 只有 driver、replay 路径和聚焦测试齐备的能力才会在矩阵中声明。 |

## 能力矩阵

每个 provider 都支持 `cloudlist` 资产枚举。资产类目包括 host / database / bucket / domain / account / log / sms / balance，按各云原生能力适配。

验证载荷覆盖：

<table>
  <tr>
    <th align="left" width="170">云厂商</th>
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
    <td align="left" width="210"><img src="icons/aws.svg" width="28" height="28" alt="AWS icon">&nbsp;<strong>AWS</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/azure.svg" width="28" height="28" alt="Azure icon">&nbsp;<strong>Azure</strong></td>
    <td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/gcp.svg" width="28" height="28" alt="GCP icon">&nbsp;<strong>GCP</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/alibaba.svg" width="28" height="28" alt="Alibaba icon">&nbsp;<strong>Alibaba</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/tencent.svg" width="28" height="28" alt="Tencent icon">&nbsp;<strong>Tencent</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/huawei.svg" width="28" height="28" alt="Huawei icon">&nbsp;<strong>Huawei</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/volcengine.svg" width="28" height="28" alt="Volcengine icon">&nbsp;<strong>Volcengine</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/jdcloud.svg" width="28" height="28" alt="JDCloud icon">&nbsp;<strong>JDCloud</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td>
  </tr>
  <tr>
    <td align="left" width="210"><img src="icons/ucloud.svg" width="28" height="28" alt="UCloud icon">&nbsp;<strong>UCloud</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td>
  </tr>
</table>


说明：`iam` = IAM 用户生命周期验证；`bucket` = 对象可见性验证；`event` = 审计日志回溯验证；`cmd` = 实例命令执行遥测验证；`rds` = 数据库账号生命周期验证；`role` = 权限绑定变更验证；`acl` = 存储公开访问验证；`cred` = 长期凭证生命周期验证。`—` 表示无原生等价能力或仍待验证。

## 快速开始

```bash
go build --ldflags "-s -w" -trimpath -o ctk cmd/main.go
./ctk                                    # 交互式 REPL
./ctk <provider> <action> [args] [flags] # 单次 headless 执行
```

在 REPL 中执行 `demo`，可让任意 provider 走内存 replay，不发起真实云调用。

## 使用边界

CloudToolKit 仅用于自有、实验室、内部或明确授权的客户环境，用来验证检测覆盖、遥测质量、调查流程和控制有效性。它不是隐蔽、绕过或未授权入侵工具，也不得用于未获授权的第三方环境。

## 文档

- [Wiki](https://github.com/404tk/cloudtoolkit/wiki) - 使用方式、payload 参考、replay walkthrough

## 致谢

- [c-bata/go-prompt](https://github.com/c-bata/go-prompt)
- [projectdiscovery/cloudlist](https://github.com/projectdiscovery/cloudlist)
