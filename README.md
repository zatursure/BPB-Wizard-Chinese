<h1 align="center">💦 BPB Wizard</h1>

A wizard to facilitate [BPB Panel](https://github.com/bia-pain-bache/BPB-Worker-Panel) deployments and management.

<p align="center">
  <img src="assets/wizard.jpg">
</p>
<br>

## 💡 Usage

> [!IMPORTANT]
> Please disconnect any Proxy or VPN before running wizard.

- You can download executable files from [Releases](https://github.com/bia-pain-bache/BPB-Wizard/releases) based on your OS, unzip and just run it.
- Android users (Termux) can use these scripts:

### ARM v8

```bash
curl -L -# -o BPB-Wizard.tar.gz https://github.com/bia-pain-bache/BPB-Wizard/releases/latest/download/BPB-Wizard-linux-arm64.tar.gz && tar xzf BPB-Wizard.tar.gz && chmod +x ./BPB-Wizard-linux-arm64 && ./BPB-Wizard-linux-arm64
```

### ARM v7 (Old models)

```bash
curl -L -# -o BPB-Wizard.tar.gz https://github.com/bia-pain-bache/BPB-Wizard/releases/latest/download/BPB-Wizard-linux-arm.tar.gz && tar xzf BPB-Wizard.tar.gz && chmod +x ./BPB-Wizard-linux-arm && ./BPB-Wizard-linux-arm
```

> [!TIP]
> 1- First it asks whether you wanna create a panel or modify an existing one and then logs you in your Cloudflare account.
>
> 2- Wizard will ask some questions for setting Panel and Configs secrets or modifying existing panels. All secrets are generated safely and randomly. However, you can use default generated values or just enter desired values.
>
> 3- In modification mode you can update your panel to the latest version or you can delete a panel.
>
> 4- if you chose creating panel, wizard opens your Panel in browser in the end! Enjoy it...
