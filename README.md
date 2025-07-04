<h1 align="center">💦 BPB 向导</h1>

本项目旨在简化 [BPB 面板](https://github.com/zatursure/BPB-Worker-Panel-Chinese) 的部署与管理流程，防止用户在部署过程中出现错误。支持 Workers 和 Pages 两种方式，强烈推荐使用。

## ⚠️ 注意事项

> [!WARNING]
> 本项目是原作者[bia-pain-bache](https://github.com/bia-pain-bache)的[BPB-Wizard](https://github.com/bia-pain-bache/BPB-Wizard)的中文翻译版本

## 💡 使用方法

### 1. Cloudflare 账号

你只需要一个 Cloudflare 账号即可使用本工具。[点击这里注册](https://dash.cloudflare.com/sign-up/)，注册后请记得查收邮件并完成账号验证。

### 2. 安装或修改 BPB 面板

> [!WARNING]
> 如果你已连接 VPN，请先断开。

#### Windows - macOS

根据你的操作系统，[下载 ZIP 文件](https://github.com/zatursure/BPB-Wizard-Chinese/releases/latest)，解压后运行程序。

#### Android (Termux) - Linux

已安装 Termux 的 Android 用户和 Linux 用户可以使用以下命令：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/zatursure/BPB-Wizard-Chinese/main/install.sh)
```

> [!IMPORTANT]  
> 请务必仅从 [官方渠道](https://github.com/termux/termux-app/releases/latest) 下载并安装 Termux。通过 Google Play 安装可能会导致问题。

第一个问题会询问你是要创建新面板还是修改账号中已有的面板。

随后会登录你的 Cloudflare 账号，返回终端后会依次询问你一系列问题。

如果选择 1，将会询问一系列配置信息。你可以直接使用默认值，也可以输入自定义值。最后会自动在浏览器中打开面板——就是这么简单。

> [!TIP]
> 每个设置项都会为你自动生成安全的专属值。你可以直接回车接受，也可以输入自己的值。

如果选择 2，会列出已部署的 Workers 和 Pages 项目，你可以选择要修改的面板。

## 面板更新

只需运行向导并在第一个问题选择 2。它会显示你账号下所有项目名称，你可以选择任意一个进行升级或删除。
