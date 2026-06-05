# autopass

自动应答交互式密码提示（密码、PIN、口令）的命令行工具。类似 `expect`，但更简单，且内置加密密钥管理。

[English](README.md)

## 为什么选 autopass？

| 工具 | 脚本 | 密钥存储 | 跨平台 | 学习成本 |
|------|------|----------|--------|----------|
| **autopass** | 无需脚本，一行命令 | AES-256-GCM，从 SSH key 派生 | Windows + Linux + macOS | 极低 |
| expect/pexpect | TCL/Python 脚本 | 无（明文写在脚本里） | 仅 Linux/macOS | 中等 |
| sshpass | 单条命令 | 明文参数或环境变量 | 仅 Linux | 低 |
| ansible vault | Playbook 级别 | 加密 vault | 通过 Ansible | 高 |

## 安装

### 下载二进制

从 [Releases](https://github.com/lifefinity/autopass/releases/latest) 下载：

```bash
# Linux (amd64)
curl -sL https://github.com/lifefinity/autopass/releases/latest/download/autopass-linux-amd64 -o autopass
chmod +x autopass && sudo mv autopass /usr/local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/lifefinity/autopass/releases/latest/download/autopass-darwin-arm64 -o autopass
chmod +x autopass && sudo mv autopass /usr/local/bin/

# Windows — 从 Releases 页面下载 autopass-windows-amd64.exe
```

### 从源码编译

```bash
git clone https://github.com/lifefinity/autopass.git
cd autopass && make build
```

## 快速开始

```bash
# 编译
make build    # → bin/autopass.exe

# 1. 添加 profile
autopass add -c "ssh user@server" -m "password:" -d "生产服务器" myserver

# 2. 运行 — 密码自动填充
autopass myserver

# 3. 查看已有配置
autopass list
```

## 工作原理

```
autopass myserver
    │
    ├─ 从 ~/.autopass/data.json 加载 profile
    ├─ 从 SSH 私钥派生 AES 密钥 (HKDF-SHA256)
    ├─ 解密存储的 secret
    ├─ 在伪终端中启动命令
    ├─ 监听输出 → 正则匹配 → 自动输入 secret
    └─ 进程正常退出
```

## 示例

### 常用 Profile

```bash
# SSH 服务器
autopass add -c "ssh deploy@prod-server" -m "password:" -d "生产部署" prod

# PostgreSQL
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" -p "=>\s*$" -d "主数据库" mydb

# MySQL
autopass add -c "mysql -h db.example.com -u root -p" -m "password:" -d "MySQL 生产" mysql-prod

# Sudo
autopass add -c "sudo apt upgrade -y" -m "password" -d "系统更新" apt-upgrade

# Kerberos
autopass add -c "kinit admin@EXAMPLE.COM" -m "password for" -d "Kerberos 认证" krb

# Midway (Amazon)
autopass add -c "mwinit -s -o" -m "PIN:" --after "date" -d "Midway 刷新" mwinit
```

### 登录后执行命令 (--then)

在密码自动填充后，在会话内执行命令（需要 `-p` 指定 shell 提示符）：

```bash
# 连接后执行 SQL
autopass mydb --then "SELECT now();" --then "\q"

# 从文件执行
autopass mydb --script queries.sql

# 组合使用
autopass mydb --then "\timing on" --script queries.sql --then "\q"
```

### 退出后执行命令 (--after)

主进程正常退出后，在新 shell 中执行命令：

```bash
# mwinit 完成后显示时间
autopass mwinit --after "date"

# SSH 退出后同步文件
autopass prod --after "rsync -a ./dist/ server:/app/"
```

### 注入环境变量 (--env / -e)

```bash
autopass deploy -e HOST=prod.example.com -e PORT=5432
```

### 更新 Profile

```bash
autopass update prod --secret                    # 更换密码
autopass update prod -c "ssh newuser@host"       # 更换命令
autopass update prod -d "新的描述"                # 更换描述
autopass update mydb --then "\timing on"         # 设置登录后步骤
autopass update mwinit --after "date"            # 设置退出后命令
autopass update mysql-prod -m "password:" -t 60s # 更换匹配和超时
```

### 静默模式

```bash
autopass mydb --quiet --script queries.sql    # 无终端输出
autopass mydb -q --then "SELECT 1;"           # 短写
```

## 命令一览

| 命令 | 说明 |
|------|------|
| `autopass <profile>` | 运行 profile，自动应答 |
| `autopass add <profile>` | 创建新 profile |
| `autopass update <profile>` | 更新 profile 字段 |
| `autopass list` | 列出所有 profile |
| `autopass remove <profile>` | 删除 profile |
| `autopass change-key <path>` | 更换加密 SSH key |
| `autopass export <file>` | 导出 profile（不含密码） |
| `autopass import <file>` | 导入 profile |
| `autopass backup <dir>` | 备份密钥和数据 |
| `autopass restore <dir>` | 恢复密钥和数据 |
| `autopass completion <shell>` | 生成 shell 补全脚本 |
| `autopass version` | 版本信息 |
| `autopass init` | 首次初始化 |

## --then 与 --after 的区别

| | `--then` | `--after` |
|---|----------|-----------|
| **何时执行** | 在运行的会话内部 | 主进程退出后 |
| **需要** | `-p` 指定 shell 提示符 | 无要求 |
| **适用场景** | psql、mysql、ssh 交互式 shell | mwinit、kinit 等一次性命令 |
| **执行环境** | PTY 会话内 | 新的 `sh -c` shell |

## 模式匹配

- **默认大小写不敏感** — `"password"` 匹配 `Password for user demo1:`、`PASSWORD:` 等
- 使用 `--case-sensitive` 启用精确匹配
- 模式是 Go 正则表达式（如 `"password|passphrase"` 匹配两者之一）
- 部分匹配即可，不需要完整行匹配

## 更换加密密钥

```bash
autopass change-key ~/.ssh/id_ed25519_new
```

用旧 key 解密所有 secret，用新 key 重新加密。两个 key 都可以有密码保护。

## 备份与恢复

```bash
autopass backup /mnt/usb/autopass-backup     # 备份
autopass restore /mnt/usb/autopass-backup    # 恢复
autopass restore ~/backup --force            # 覆盖现有数据
```

导出/导入（不含密钥，适合共享配置）：

```bash
autopass export profiles.json                # 导出（不含 secret）
autopass import profiles.json                # 导入合并
autopass import profiles.json --force        # 覆盖同名
```

## Shell 补全

```bash
# Bash — 加入 ~/.bashrc
eval "$(autopass completion bash)"

# Zsh — 加入 ~/.zshrc
eval "$(autopass completion zsh)"

# Fish
autopass completion fish > ~/.config/fish/completions/autopass.fish
```

支持 Tab 补全 profile 名称。

## 安全性

- 使用 **AES-256-GCM** 加密（每个 secret 独立随机 nonce）
- 加密密钥从 **SSH 私钥** 通过 HKDF-SHA256 派生，不存储在磁盘上
- 如果没有 SSH key，自动生成 `~/.autopass/autopass_key`（ed25519）
- 数据文件 `~/.autopass/data.json` 权限为 0600
- 任何地方都没有明文 secret

## 平台支持

| 平台 | 方式 |
|------|------|
| Windows 10+ | ConPTY |
| Linux | PTY (creack/pty) |
| macOS | PTY (creack/pty) |

## 文档

- [用户指南](docs/user-guide.md) — 详细用法、标志、故障排除
- [架构设计](docs/architecture.md) — 组件设计、数据流、安全模型
- [开发指南](docs/development.md) — 编译、测试、贡献
