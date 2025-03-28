package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

type KVNamespaceBinding struct {
	Binding string `json:"binding"`
	Id      string `json:"id"`
}

type KVNamespace struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

type Routes struct {
	Custom_domain bool   `json:"custom_domain"`
	Pattern       string `json:"pattern"`
}

type WranglerConfig struct {
	Name                string               `json:"name"`
	Main                string               `json:"main"`
	Compatibility_date  string               `json:"compatibility_date"`
	Compatibility_flags []string             `json:"compatibility_flags"`
	Workers_dev         bool                 `json:"workers_dev"`
	Kv_namespaces       []KVNamespaceBinding `json:"kv_namespaces"`
	Vars                workerSettings       `json:"vars"`
	Routes              []Routes             `json:"routes,omitempty"`
}

type workerSettings struct {
	Uuid     string `json:"UUID"`
	TrPass   string `json:"TR_PASS"`
	ProxyIP  string `json:"PROXY_IP"`
	Fallback string `json:"FALLBACK"`
	SubPath  string `json:"SUB_PATH"`
}

var (
	kvID         string
	workerName   string
	customDomain string
	UUID         string
	TR_PASS      string
	PROXY_IP     string
	FALLBACK     string
	SUB_PATH     string
	red          = "\033[31m"
	green        = "\033[32m"
	reset        = "\033[0m"
	orange       = "\033[38;2;255;165;0m"
	blue         = "\033[94m"
	bold         = "\033[1m"
	title        = bold + blue + "●" + reset
	ask          = bold + "-" + reset
	info         = bold + "+" + reset
)

//go:embed bundles/node-v22.14.0-win-x64.zip
var embeddedNodeZip []byte

//go:embed bundles/wrangler.zip
var embeddedWranglerZip []byte

func main() {

	if runtime.GOOS != "windows" {
		failMessage("Unsupported OS. This script is for Windows only.", nil)
		return
	}

	installDir := filepath.Join(os.Getenv("USERPROFILE"), ".bpb-wizard")
	nodeDir := filepath.Join(installDir, "node-v22.14.0-win-x64")
	wranglerConfigPath := filepath.Join(installDir, "wrangler.json")
	nodeZipPath := filepath.Join(os.TempDir(), "node-v22.14.0-win-x64.zip")
	wranglerZipPath := filepath.Join(os.TempDir(), "wrangler.zip")
	workerURL := "https://github.com/bia-pain-bache/BPB-Worker-Panel/releases/latest/download/worker.js"
	workerPath := filepath.Join(installDir, "worker.js")

	if _, err := os.Stat(wranglerConfigPath); !errors.Is(err, os.ErrNotExist) {
		if err := os.Remove(wranglerConfigPath); err != nil {
			failMessage("Error deleting old worker config.", err)
			return
		}
	}

	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("%s;%s", nodeDir, currentPath)
	if err := os.Setenv("PATH", newPath); err != nil {
		failMessage("Error setting PATH environment variable", err)
		return
	}
	fmt.Printf("\n%s Installing %sBPB Wizard%s...\n", title, blue, reset)

	if _, err := runCommand(installDir, "npx", "wrangler", "-v"); err != nil {
		if err := os.WriteFile(nodeZipPath, embeddedNodeZip, 0644); err != nil {
			failMessage("Error copying Node.js", err)
			return
		}

		if err := os.WriteFile(wranglerZipPath, embeddedWranglerZip, 0644); err != nil {
			failMessage("Error copying Wrangler", err)
			return
		}

		successMessage("Files copied successfuly.")
		if err := unzip(nodeZipPath, installDir); err != nil {
			failMessage("Error extracting Node.js", err)
			return
		}

		if err := unzip(wranglerZipPath, installDir); err != nil {
			failMessage("Error extracting Wrangler", err)
			return
		}
		successMessage("Dependencies installed successfully!")
	} else {
		successMessage("BPB Wizard is already installed!")
	}

	fmt.Printf("\n%s Downloading %sworker.js%s...\n", title, green, reset)
	for {
		if err := downloadFile(workerURL, workerPath); err != nil {
			failMessage("Error downloading worker.js", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}
		successMessage("Worker downloaded successfully!")
		break
	}

	fmt.Printf("\n%s Login %sCloudflare%s...\n", title, orange, reset)
	for {
		if _, err := runCommand(installDir, "npx", "wrangler", "login"); err != nil {
			failMessage("Error logging into Cloudflare", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		if _, err := runCommand(installDir, "npx", "wrangler", "telemetry", "disable"); err != nil {
			failMessage("Error disabling telemetry.", err)
			return
		}

		successMessage("Cloudflare logged in successfully!")
		break
	}

	fmt.Printf("\n%s Get Worker settings...\n", title)
	var prompt string
	for {
		workerName = generateRandomDomain(32)
		fmt.Printf("\n%s The random generated worker name (%sSubdomain%s) is: %s%s%s\n", info, green, reset, orange, workerName, reset)
		if response := promptUser("Please enter a custom worker name or press ENTER to use generated one: "); response != "" {
			if strings.Contains(strings.ToLower(response), "bpb") {
				message := fmt.Sprintf("Worker name cannot contain %sbpb%s! Please try another name.", red, reset)
				failMessage(message, nil)
				continue
			}
			workerName = response
		}

		if resp := isWorkerAvailable(installDir, workerName); resp {
			prompt = fmt.Sprintf("This worker already exists! This will %sRESET%s all panel settings, would you like to override it? (y/n): ", red, reset)
			if response := promptUser(prompt); strings.ToLower(response) == "n" {
				continue
			}
		}
		break
	}

	UUID = uuid.NewString()
	fmt.Printf("\n%s The random generated %sUUID%s is: %s%s%s\n", info, green, reset, orange, UUID, reset)
	if response := promptUser("Please enter a custom UUID or press ENTER to use generated one: "); response != "" {
		UUID = response
	}

	TR_PASS = generateTrPassword(12)
	fmt.Printf("\n%s The random generated %sTrojan password%s is: %s%s%s\n", info, green, reset, orange, TR_PASS, reset)
	if response := promptUser("Please enter a custom Trojan password or press ENTER to use generated one: "); response != "" {
		TR_PASS = response
	}

	PROXY_IP = "bpb.yousef.isegaro.com"
	fmt.Printf("\n%s The default %sProxy IP%s is: %s%s%s\n", info, green, reset, orange, PROXY_IP, reset)
	if response := promptUser("Please enter custom Proxy IP/Domains or press ENTER to use default: "); response != "" {
		PROXY_IP = response
	}

	FALLBACK = "speed.cloudflare.com"
	fmt.Printf("\n%s The default %sFallback domain%s is: %s%s%s\n", info, green, reset, orange, FALLBACK, reset)
	if response := promptUser("Please enter a custom Fallback domain or press ENTER to use default: "); response != "" {
		FALLBACK = response
	}

	SUB_PATH = generateSubURIPath(16)
	fmt.Printf("\n%s The random generated %sSubscription path%s is: %s%s%s\n", info, green, reset, orange, SUB_PATH, reset)
	if response := promptUser("Please enter a custom Subscription path or press ENTER to use generated one: "); response != "" {
		SUB_PATH = response
	}

	fmt.Printf("\n%s You can set %sCustom domain%s ONLY if you registered domain on this cloudflare account.\n", info, green, reset)
	if response := promptUser("Please enter a custom domain (if you have any) or press ENTER to ignore: "); response != "" {
		customDomain = response
	}

	fmt.Printf("\n%s Creating KV namespace...\n", title)
	for {
		now := time.Now().Format("2006-01-02_15-04-05")
		kvName := fmt.Sprintf("panel-kv-%s", now)
		output, err := runCommand(installDir, "npx", "wrangler", "kv", "namespace", "create", kvName)
		if err != nil {
			failMessage("Error creating KV!", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		id, err := extractKvID(output)
		if err != nil {
			failMessage("Error getting KV ID", err)
			return
		}

		kvID = id
		break
	}
	successMessage("KV created successfully!")

	fmt.Printf("\n%s Building worker configuration...\n", title)
	if err := buildWranglerConfig(wranglerConfigPath); err != nil {
		failMessage("Error building Wrangler configuration", err)
		return
	}
	successMessage("Worker configuration built successfully!")

	for {
		fmt.Printf("\n%s Deploying worker...\n", title)
		output, err := runCommand(installDir, "npx", "wrangler", "deploy")
		if err != nil {
			failMessage("Error deploying worker", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}
		successMessage("Worker deployed successfully!")

		prompt := fmt.Sprintf("Would you like to open %sBPB panel%s in browser? (y/n): ", blue, reset)
		if response := promptUser(prompt); strings.ToLower(response) == "n" {
			return
		}

		var panelURL string
		if customDomain == "" {
			url, err := extractURL(output)
			if err != nil {
				failMessage("Error getting URL", err)
				return
			}
			panelURL = url + "/panel"
		} else {
			panelURL = "https://" + customDomain + "/panel"
		}

		if err := openURL(panelURL); err != nil {
			failMessage("Error opening URL in browser", err)
			return
		}
		break
	}
}

func runCommand(cmdDir string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Dir = cmdDir
	err := cmd.Run()
	output := stdoutBuf.String() + stderrBuf.String()
	if err != nil {
		return output, err
	}

	return output, nil
}

func promptUser(prompt string) string {
	fmt.Printf("%s %s", ask, prompt)
	var response string
	fmt.Scanln(&response)
	return strings.TrimSpace(response)
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

func generateRandomDomain(subDomainLength int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789-"
	return generateRandomString(charset, subDomainLength, true)
}

func generateTrPassword(passwordLength int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+[]{}|;:',.<>?"
	return generateRandomString(charset, passwordLength, false)
}

func generateSubURIPath(uriLength int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@$&*_-+;:,."
	return generateRandomString(charset, uriLength, false)
}

func isWorkerAvailable(installDir string, workerName string) bool {
	if _, err := runCommand(installDir, "npx", "wrangler", "deployments", "list", "--name", workerName); err == nil {
		return true
	}

	return false
}

func extractURL(output string) (string, error) {
	url, err := findMatch(`https?://[^\s]+`, output)
	if err != nil {
		return "", fmt.Errorf("failed to get panel URL: %v", err)
	}

	return url, nil
}

func buildWranglerConfig(filePath string) error {
	config := WranglerConfig{
		Name:                workerName,
		Main:                "./worker.js",
		Compatibility_date:  time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		Compatibility_flags: []string{"nodejs_compat"},
		Workers_dev:         true,
		Kv_namespaces: []KVNamespaceBinding{
			{Binding: "kv", Id: kvID},
		},
		Vars: workerSettings{
			Uuid:     UUID,
			TrPass:   TR_PASS,
			ProxyIP:  PROXY_IP,
			Fallback: FALLBACK,
			SubPath:  SUB_PATH,
		},
		Routes: []Routes{
			{Custom_domain: true, Pattern: customDomain},
		},
	}

	if customDomain == "" {
		config.Routes = nil
	}

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config to JSON: %v", err)
	}

	if err = os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing JSON to file: %v", err)
	}

	return nil
}

func findMatch(pattern string, input string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindAllString(input, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no matches found")
	}
	return matches[len(matches)-1], nil
}

func extractKvID(output string) (string, error) {
	jsonPart, err := findMatch(`\{[^{}]*\}`, output)
	if err != nil {
		return "", fmt.Errorf("failed to parse KV object: %v", err)
	}

	var object KVNamespace
	if err := json.Unmarshal([]byte(jsonPart), &object); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return object.ID, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error making GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s (HTTP %d)", url, resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}

	return nil
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fPath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fPath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func openURL(url string) error {
	var cmd *exec.Cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return cmd.Start()
}

func failMessage(message string, err error) {
	errMark := bold + red + "✗" + reset
	if err != nil {
		fmt.Printf("%s %s: %s\n", errMark, message, err)
	}

	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	succMark := bold + green + "✓" + reset
	fmt.Printf("%s %s\n", succMark, message)
}
