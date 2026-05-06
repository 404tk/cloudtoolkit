# CloudToolKit

<a href="./README.md">English</a> | <strong>简体中文</strong>

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
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="AWS"><rect width="24" height="24" rx="5" fill="#232F3E"/><text x="12" y="11" text-anchor="middle" font-family="Arial, sans-serif" font-size="7" font-weight="700" fill="#fff">aws</text><path d="M6 15c3 2.5 9 2.5 12 0" fill="none" stroke="#FF9900" stroke-width="1.8" stroke-linecap="round"/></svg>&nbsp;<strong>AWS</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="Azure"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><path d="M7 18 11 5h4l-3 7-2 6H7z" fill="#0078D4"/><path d="M15 5 19 18h-7l-2-4 3-4-1 2z" fill="#50A7F8"/></svg>&nbsp;<strong>Azure</strong></td>
    <td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="GCP"><rect width="24" height="24" rx="5" fill="#fff" stroke="#DCE5F0"/><path d="M7 15h10" stroke="#4285F4" stroke-width="2.2" stroke-linecap="round"/><path d="M7 15a5 5 0 0 1 2-9" stroke="#34A853" stroke-width="2.2" stroke-linecap="round" fill="none"/><path d="M9 6a6 6 0 0 1 7 1" stroke="#FBBC04" stroke-width="2.2" stroke-linecap="round" fill="none"/><path d="M16 7a5 5 0 0 1 1 8" stroke="#EA4335" stroke-width="2.2" stroke-linecap="round" fill="none"/></svg>&nbsp;<strong>GCP</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="Alibaba"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><g fill="none" stroke="#FF6A00" stroke-width="2" stroke-linecap="round"><path d="M8 7h3M8 7v10M8 17h3M16 7h-3M16 7v10M16 17h-3M10 12h4"/></g></svg>&nbsp;<strong>Alibaba</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="Tencent"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><path d="M7 16h10a4 4 0 0 0 0-8h-1a6 6 0 0 0-11 3 4 4 0 0 0 2 5z" fill="none" stroke="#1E9BFF" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>&nbsp;<strong>Tencent</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="Huawei"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><g fill="#E53935" transform="translate(12 10)"><ellipse rx="1.4" ry="4.5" transform="rotate(-55) translate(0 -2.5)"/><ellipse rx="1.4" ry="4.5" transform="rotate(-30) translate(0 -2.5)"/><ellipse rx="1.4" ry="4.5" transform="rotate(0) translate(0 -2.5)"/><ellipse rx="1.4" ry="4.5" transform="rotate(30) translate(0 -2.5)"/><ellipse rx="1.4" ry="4.5" transform="rotate(55) translate(0 -2.5)"/></g><path d="M8 16c2 1.8 6 1.8 8 0" fill="none" stroke="#E53935" stroke-width="1.8" stroke-linecap="round"/></svg>&nbsp;<strong>Huawei</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="Volcengine"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><path d="M5 17 10 6l3 7 3-4 4 8h-4l-3-4-2 4z" fill="#2B6CFF"/></svg>&nbsp;<strong>Volcengine</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td><td align="center">✓</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="JDCloud"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><path d="M7 16h10a4 4 0 0 0 0-8h-1a6 6 0 0 0-11 3 4 4 0 0 0 2 5z" fill="none" stroke="#F44336" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M11 14c0 3-1.5 4.5-4 4.5" fill="none" stroke="#F44336" stroke-width="1.8" stroke-linecap="round"/></svg>&nbsp;<strong>JDCloud</strong></td>
    <td align="center">✓</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">—</td><td align="center">✓</td><td align="center">✓</td><td align="center">—</td>
  </tr>
  <tr>
    <td align="left" width="170"><svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" role="img" aria-label="UCloud"><rect width="24" height="24" rx="5" fill="#F8FAFC" stroke="#DCE5F0"/><path d="M7 6v8c0 3 2 5 5 5s5-2 5-5V6" fill="none" stroke="#2B6CFF" stroke-width="2.5" stroke-linecap="round"/><circle cx="17" cy="7" r="1.6" fill="#2B6CFF"/></svg>&nbsp;<strong>UCloud</strong></td>
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
