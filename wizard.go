package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go/v4/kv"
	"github.com/google/uuid"
)

type DeployType int

const (
	DTWorker DeployType = iota
	DTPage
)

var DeployTypeNames = map[DeployType]string{
	DTWorker: "worker",
	DTPage:   "page",
}

func (dt DeployType) String() string {
	return DeployTypeNames[dt]
}

type Panel struct {
	Name string
	Type string
}

const (
	CharsetAlphaNumeric      = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	CharsetSpecialCharacters = "!@#$%^&*()_+[]{}|;:',.<>?"
	CharsetTrojanPassword    = CharsetAlphaNumeric + CharsetSpecialCharacters
	CharsetSubDomain         = "abcdefghijklmnopqrstuvwxyz0123456789-"
	CharsetURIPath           = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@$&*_-+;:,."
	DomainRegex              = `^(?i)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`
)

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading worker.js: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func downloadWorker() error {
	fmt.Printf("\n%s Downloading %s...\n", title, fmtStr("worker.js", GREEN, true))

	for {
		if _, err := os.Stat(workerPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to check worker.js: %w", err)
			}
		} else {
			successMessage("worker.js already exists, skipping download.")
			return nil
		}

		if err := downloadFile(workerURL, workerPath); err != nil {
			failMessage("Failed to download worker.js")
			log.Printf("%v\n", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				os.Exit(0)
			}
			continue
		}

		successMessage("worker.js downloaded successfully!")
		return nil
	}
}

func generateRandomString(charSet string, length int, isDomain bool) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomBytes := make([]byte, length)

	for i := range randomBytes {
		for {
			char := charSet[r.Intn(len(charSet))]
			if isDomain && (i == 0 || i == length-1) && char == byte('-') {
				continue
			}
			randomBytes[i] = char
			break
		}
	}

	return string(randomBytes)
}

func generateRandomSubDomain(subDomainLength int) string {
	return generateRandomString(CharsetSubDomain, subDomainLength, true)
}

func isValidSubDomain(subDomain string) error {
	if strings.Contains(subDomain, "bpb") {
		message := fmt.Sprintf("Name cannot contain %s. Please try again.\n", fmtStr("bpb", RED, true))
		return fmt.Errorf("%s", message)
	}

	subdomainRegex := regexp.MustCompile(`^(?i)[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	isValid := subdomainRegex.MatchString(subDomain)
	if !isValid {
		message := fmt.Sprintf("Subdomain cannot start with %s and should only contain %s and %s. Please try again.\n", fmtStr("-", RED, true), fmtStr("A-Z", GREEN, true), fmtStr("0-9", GREEN, true))
		return fmt.Errorf("%s", message)
	}
	return nil
}

func isValidIpDomain(value string) bool {
	if net.ParseIP(value) != nil {
		return true
	}

	domainRegex := regexp.MustCompile(DomainRegex)
	return domainRegex.MatchString(value)
}

func isValidHost(value string) bool {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return false
	}

	domainRegex := regexp.MustCompile(DomainRegex)
	if net.ParseIP(host) == nil && !domainRegex.MatchString(host) {
		return false
	}

	intPort, err := strconv.Atoi(port)
	if err != nil || intPort < 1 || intPort > 65535 {
		return false
	}

	return true
}

func generateTrPassword(passwordLength int) string {
	return generateRandomString(CharsetTrojanPassword, passwordLength, false)
}

func isValidTrPassword(trojanPassword string) bool {
	for _, c := range trojanPassword {
		if !strings.ContainsRune(CharsetTrojanPassword, c) {
			return false
		}
	}

	return true
}

func generateSubURIPath(uriLength int) string {
	return generateRandomString(CharsetURIPath, uriLength, false)
}

func isValidSubURIPath(uri string) bool {
	for _, c := range uri {
		if !strings.ContainsRune(CharsetURIPath, c) {
			return false
		}
	}

	return true
}

func promptUser(prompt string) string {
	fmt.Printf("%s %s", ask, prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("\n%s Exiting...\n", title)
		if err == io.EOF {
			os.Exit(0)
		}
		os.Exit(1)
	}

	return strings.TrimSpace(input)
}

func failMessage(message string) {
	errMark := fmtStr("✗", RED, true)
	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	succMark := fmtStr("✓", GREEN, true)
	fmt.Printf("%s %s\n", succMark, message)
}

func openURL(url string) error {
	var cmd string
	var args = []string{url}

	switch runtime.GOOS {
	case "darwin": // MacOS
		cmd = "open"
	case "windows": // Windows
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // Linux, BSD, Android, etc.
		if isAndroid {
			termuxBin := os.Getenv("PATH")
			cmd = filepath.Join(termuxBin, "termux-open-url")
		} else {
			cmd = "xdg-open"
		}
	}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		return err
	}

	return nil
}

func checkBPBPanel(url string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(5000) * time.Millisecond,
				}

				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext:       dialContext,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	for range ticker.C {
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf(".")
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf(".")
			resp.Body.Close()
			continue
		}

		resp.Body.Close()
		message := fmt.Sprintf("BPB panel is ready -> %s", url)
		successMessage(message)
		fmt.Print("\n")
		prompt := fmt.Sprintf("Would you like to open %s in browser? (y/n): ", fmtStr("BPB panel", BLUE, true))

		if response := promptUser(prompt); strings.ToLower(response) == "n" {
			return nil
		}

		if err = openURL(url); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func runWizard() {
	renderHeader()
	fmt.Printf("\n%s 欢迎使用 %s！\n", title, fmtStr("BPB 向导", GREEN, true))
	fmt.Printf("%s 本向导将帮助你在 Cloudflare 上部署或修改 %s。\n", info, fmtStr("BPB 面板", BLUE, true))
	fmt.Printf("%s 请确保你拥有已验证的 %s 账号。\n\n", info, fmtStr("Cloudflare", ORANGE, true))

	for {
		message := fmt.Sprintf("请输入 1 以%s新面板，或 2 以%s已有面板: ", fmtStr("创建", GREEN, true), fmtStr("修改", RED, true))
		response := promptUser(message)
		switch response {
		case "1":
			createPanel()
		case "2":
			modifyPanel()
		default:
			failMessage("选择错误，请只输入 1 或 2！\n")
			continue
		}

		res := promptUser("是否再次运行向导？(y/n): ")
		if strings.ToLower(res) == "n" {
			fmt.Printf("\n%s 退出...\n", title)
			return
		}
	}
}

func createPanel() {
	ctx := context.Background()
	var err error
	if cfClient == nil || cfAccount == nil {
		go login()
		token := <-obtainedToken
		cfClient = NewClient(token)

		cfAccount, err = getAccount(ctx)
		if err != nil {
			failMessage("获取 Cloudflare 账号失败。")
			log.Fatalln(err)
		}
	}

	fmt.Printf("\n%s 获取设置...\n", title)
	fmt.Printf("\n%s 你可以选择使用 %s 或 %s 进行部署。\n", info, fmtStr("Workers", ORANGE, true), fmtStr("Pages", ORANGE, true))
	fmt.Printf("%s %s: 如果选择 %s，访问面板可能需要最多 5 分钟，请耐心等待！\n", info, warning, fmtStr("Pages", ORANGE, true))
	var deployType DeployType

	for {
		response := promptUser("请输入 1 选择 Workers 或 2 选择 Pages 部署: ")
		switch response {
		case "1":
			deployType = DTWorker
		case "2":
			deployType = DTPage
		default:
			failMessage("选择错误，请只输入 1 或 2！")
			continue
		}

		break
	}

	var projectName string
	for {
		projectName = generateRandomSubDomain(32)
		fmt.Printf("\n%s 随机生成的名称（%s）为: %s\n", info, fmtStr("子域名", GREEN, true), fmtStr(projectName, ORANGE, true))
		if response := promptUser("请输入自定义名称或直接回车使用生成的名称: "); response != "" {
			if err := isValidSubDomain(response); err != nil {
				failMessage(err.Error())
				continue
			}

			projectName = response
		}

		var isAvailable bool
		fmt.Printf("\n%s 检查域名可用性...\n", title)

		if deployType == DTWorker {
			isAvailable = isWorkerAvailable(ctx, projectName)
		} else {
			isAvailable = isPagesProjectAvailable(ctx, projectName)
		}

		if !isAvailable {
			prompt := fmt.Sprintf("该名称已存在！这将%s所有面板设置，是否覆盖？(y/n): ", fmtStr("重置", RED, true))
			if response := promptUser(prompt); strings.ToLower(response) == "n" {
				continue
			}
		}

		successMessage("可用！")
		break
	}

	uid := uuid.NewString()
	fmt.Printf("\n%s 随机生成的 %s 为: %s\n", info, fmtStr("UUID", GREEN, true), fmtStr(uid, ORANGE, true))
	for {
		if response := promptUser("请输入自定义 uid 或直接回车使用生成的 uid: "); response != "" {
			if _, err := uuid.Parse(response); err != nil {
				failMessage("UUID 非标准格式，请重试。\n")
				continue
			}

			uid = response
		}

		break
	}

	trPass := generateTrPassword(12)
	fmt.Printf("\n%s 随机生成的 %s 为: %s\n", info, fmtStr("Trojan 密码", GREEN, true), fmtStr(trPass, ORANGE, true))
	for {
		if response := promptUser("请输入自定义 Trojan 密码或直接回车使用生成的密码: "); response != "" {
			if !isValidTrPassword(response) {
				failMessage("Trojan 密码不能包含非标准字符！请重试。\n")
				continue
			}

			trPass = response
		}

		break
	}

	proxyIP := "bpb.yousef.isegaro.com"
	fmt.Printf("\n%s 默认 %s 为: %s\n", info, fmtStr("代理 IP", GREEN, true), fmtStr(proxyIP, ORANGE, true))
	for {
		if response := promptUser("请输入自定义代理 IP/域名，或直接回车使用默认值: "); response != "" {
			areValid := true
			values := strings.SplitSeq(response, ",")
			for v := range values {
				trimmedValue := strings.TrimSpace(v)
				if !isValidIpDomain(trimmedValue) && !isValidHost(trimmedValue) {
					areValid = false
					message := fmt.Sprintf("%s 不是有效的 IP 或域名，请重试。", trimmedValue)
					failMessage(message)
				}
			}

			if !areValid {
				continue
			}

			proxyIP = response
		}

		break
	}

	fallback := "speed.cloudflare.com"
	fmt.Printf("\n%s 默认 %s 为: %s\n", info, fmtStr("回落域名", GREEN, true), fmtStr(fallback, ORANGE, true))
	if response := promptUser("请输入自定义回落域名或直接回车使用默认值: "); response != "" {
		fallback = response
	}

	subPath := generateSubURIPath(16)
	fmt.Printf("\n%s 随机生成的 %s 为: %s\n", info, fmtStr("订阅路径", GREEN, true), fmtStr(subPath, ORANGE, true))
	for {
		if response := promptUser("请输入自定义订阅路径或直接回车使用生成的路径: "); response != "" {
			if !isValidSubURIPath(response) {
				failMessage("URI 不能包含非标准字符！请重试。\n")
				continue
			}

			subPath = response
		}

		break
	}

	var customDomain string
	fmt.Printf("\n%s 仅当你在本 Cloudflare 账号下注册了域名时，才可设置 %s。\n", info, fmtStr("自定义域名", GREEN, true))
	if response := promptUser("请输入自定义域名（如有）或直接回车跳过: "); response != "" {
		customDomain = response
	}

	fmt.Printf("\n%s 创建 KV 命名空间...\n", title)
	var kvNamespace *kv.Namespace

	for {
		now := time.Now().Format("2006-01-02_15-04-05")
		kvName := fmt.Sprintf("panel-kv-%s", now)
		kvNamespace, err = createKVNamespace(ctx, kvName)
		if err != nil {
			failMessage("创建 KV 失败。")
			log.Printf("%v\n\n", err)
			if response := promptUser("是否重试？(y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		successMessage("KV 创建成功！")
		break
	}

	var panel string
	if err := downloadWorker(); err != nil {
		failMessage("下载 worker.js 失败")
		log.Fatalln(err)
	}

	switch deployType {
	case DTWorker:
		panel, err = deployWorker(ctx, projectName, uid, trPass, proxyIP, fallback, subPath, kvNamespace, customDomain)
	case DTPage:
		panel, err = deployPagesProject(ctx, projectName, uid, trPass, proxyIP, fallback, subPath, kvNamespace, customDomain)
	}

	if err != nil {
		failMessage("获取面板 URL 失败。")
		log.Fatalln(err)
	}

	if err := checkBPBPanel(panel); err != nil {
		failMessage("检测 BPB 面板失败。")
		log.Fatalln(err)
	}
}

func modifyPanel() {
	ctx := context.Background()
	var err error
	if cfClient == nil || cfAccount == nil {
		go login()
		token := <-obtainedToken
		cfClient = NewClient(token)

		cfAccount, err = getAccount(ctx)
		if err != nil {
			failMessage("获取 Cloudflare 账号失败。")
			log.Fatalln(err)
		}
	}

	for {
		var panels []Panel
		var message string

		fmt.Printf("\n%s 获取面板列表...\n", title)
		workersList, err := listWorkers(ctx)
		if err != nil {
			failMessage("获取 workers 列表失败。")
			log.Println(err)
		} else {
			for _, worker := range workersList {
				panels = append(panels, Panel{
					Name: worker,
					Type: "workers",
				})
			}
		}

		pagesList, err := listPages(ctx)
		if err != nil {
			failMessage("获取 pages 列表失败。")
			log.Println(err)
		} else {
			for _, pages := range pagesList {
				panels = append(panels, Panel{
					Name: pages,
					Type: "pages",
				})
			}
		}

		if len(panels) == 0 {
			failMessage("未找到 Workers 或 Pages，正在退出...")
			return
		}

		message = fmt.Sprintf("共找到 %d 个 workers 和 pages 项目:\n", len(panels))
		successMessage(message)
		for i, panel := range panels {
			fmt.Printf(" %s %s - %s\n", fmtStr(strconv.Itoa(i+1)+".", BLUE, true), panel.Name, fmtStr(panel.Type, ORANGE, true))
		}

		var index int
		for {
			fmt.Println("")
			response := promptUser("请选择你要修改的编号: ")
			index, err = strconv.Atoi(response)
			if err != nil || index < 1 || index > len(panels) {
				failMessage("选择无效，请重试。")
				continue
			}

			break
		}

		panelName := panels[index-1].Name
		panelType := panels[index-1].Type

		message = fmt.Sprintf("请输入 1 以%s面板，或 2 以%s面板: ", fmtStr("更新", GREEN, true), fmtStr("删除", RED, true))
		response := promptUser(message)
		for {
			switch response {
			case "1":

				if err := downloadWorker(); err != nil {
					failMessage("下载 worker.js 失败")
					log.Fatalln(err)
				}

				if panelType == "workers" {
					if err := updateWorker(ctx, panelName); err != nil {
						failMessage("更新面板失败。")
						log.Fatalln(err)
					}

					successMessage("面板更新成功！\n")
					break
				}

				if err := updatePagesProject(ctx, panelName); err != nil {
					failMessage("更新面板失败。")
					log.Fatalln(err)
				}

				successMessage("面板更新成功！\n")

			case "2":

				if panelType == "workers" {
					if err := deleteWorker(ctx, panelName); err != nil {
						failMessage("删除面板失败。")
						log.Fatalln(err)
					}

					successMessage("面板删除成功！\n")
					break
				}

				if err := deletePagesProject(ctx, panelName); err != nil {
					failMessage("删除面板失败。")
					log.Fatalln(err)
				}

				successMessage("面板删除成功！\n")

			default:
				failMessage("选择错误，请只输入 1 或 2！")
				continue
			}

			break
		}

		if response := promptUser("是否继续修改其他面板？(y/n): "); strings.ToLower(response) == "n" {
			break
		}
	}
}
