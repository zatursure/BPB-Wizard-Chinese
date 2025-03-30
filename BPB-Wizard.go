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

type KVNamespace struct {
	Title string `json:"title"`
	ID    string `json:"id"`
}

var (
	kvID         string
	projectName  string
	customDomain string
	deployType   string
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
	srsPath := filepath.Join(installDir, "src")

	if _, err := os.Stat(wranglerConfigPath); !errors.Is(err, os.ErrNotExist) {
		if err := os.Remove(wranglerConfigPath); err != nil {
			failMessage("Error deleting old worker config.", err)
			return
		}
	}

	if err := os.RemoveAll(srsPath); err != nil {
		failMessage("Error deleting old worker.js file.", err)
	}

	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("%s;%s", nodeDir, currentPath)
	if err := os.Setenv("PATH", newPath); err != nil {
		failMessage("Error setting PATH environment variable", err)
		return
	}
	fmt.Printf("\n%s Installing %sBPB Wizard%s...\n", title, blue, reset)

	if _, err := runCommand(installDir, "npx wrangler -v"); err != nil {
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

	fmt.Printf("\n%s Login %sCloudflare%s...\n", title, orange, reset)
	for {
		if _, err := runCommand(installDir, "npx wrangler login"); err != nil {
			failMessage("Error logging into Cloudflare", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		if _, err := runCommand(installDir, "npx wrangler telemetry disable"); err != nil {
			failMessage("Error disabling telemetry.", err)
			return
		}

		successMessage("Cloudflare logged in successfully!")
		break
	}

	fmt.Printf("\n%s Get Worker settings...\n", title)

	fmt.Printf("\n%s You can use %sWorkers%s or %sPages%s to deploy.\n", info, green, reset, green, reset)
	fmt.Printf("%s %sWarning%s: If you choose %sPages%s, you can not modify settings like UUID from Cloudflare dashboard later, you have to modify it from here.\n", info, red, reset, green, reset)
	fmt.Printf("%s %sWarning%s: If you choose %sPages%s, sometimes it takes about 5 minutes until you can open panel, so please keep calm!\n", info, red, reset, green, reset)
	for {
		response := promptUser("Please enter 1 for Workers or 2 for Pages deployment: ")
		if !(response == "1" || response == "2") {
			failMessage("Wrong selection, Please choose 1 or 2 only!", nil)
			continue
		}

		deployType = response
		break
	}

	for {
		projectName = generateRandomDomain(32)
		fmt.Printf("\n%s The random generated worker name (%sSubdomain%s) is: %s%s%s\n", info, green, reset, orange, projectName, reset)
		if response := promptUser("Please enter a custom worker name or press ENTER to use generated one: "); response != "" {
			if strings.Contains(strings.ToLower(response), "bpb") {
				message := fmt.Sprintf("Worker name cannot contain %sbpb%s! Please try another name.", red, reset)
				failMessage(message, nil)
				continue
			}
			projectName = response
		}

		fmt.Printf("\n%s Checking domain availablity...\n", title)
		if resp := isWorkerAvailable(installDir, projectName, deployType); resp {
			prompt := fmt.Sprintf("This worker already exists! This will %sRESET%s all panel settings, would you like to override it? (y/n): ", red, reset)
			if response := promptUser(prompt); strings.ToLower(response) == "n" {
				continue
			}
		}
		successMessage("Available!")
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

	fmt.Printf("\n%s Downloading %sworker.js%s...\n", title, green, reset)
	if err := os.Mkdir(srsPath, 0750); err != nil {
		failMessage("Could not create src directory", err)
		return
	}

	var workerPath = filepath.Join(srsPath, "worker.js")
	if deployType == "2" {
		workerPath = filepath.Join(srsPath, "_worker.js")
	}
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

	fmt.Printf("\n%s Creating KV namespace...\n", title)
	for {
		now := time.Now().Format("2006-01-02_15-04-05")
		kvName := fmt.Sprintf("panel-kv-%s", now)
		output, err := runCommand(installDir, "npx wrangler kv namespace create "+kvName)
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

	fmt.Printf("\n%s Building panel configuration...\n", title)
	if err := buildWranglerConfig(wranglerConfigPath); err != nil {
		failMessage("Error building Wrangler configuration", err)
		return
	}
	successMessage("Panel configuration built successfully!")

	for {
		fmt.Printf("\n%s Deploying %sBPB Panel%s...\n", title, blue, reset)
		if deployType == "1" {
			output, err := runCommand(installDir, "npx wrangler deploy")
			if err != nil {
				failMessage("Error deploying Panel", err)
				if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
					return
				}
				continue
			}

			successMessage("Panel deployed successfully!")
			launchPanel(output)
			break
		}

		if _, err := runCommand(installDir, "npx wrangler pages project create "+projectName+" --production-branch production"); err != nil {
			failMessage("Error creating Pages project", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		_, err := runCommand(installDir, "npx wrangler pages deploy --commit-dirty true --branch production")
		if err != nil {
			failMessage("Error deploying Panel", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return
			}
			continue
		}

		successMessage("Panel deployed successfully!")
		launchPanel("")
		break
	}
}

func runCommand(cmdDir string, command string) (string, error) {
	c := strings.Split(command, " ")
	cmd := exec.Command(c[0], c[1:]...)
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

func isWorkerAvailable(installDir, projectName, deployType string) bool {
	var command string
	if deployType == "1" {
		command = "npx wrangler deployments list --name " + projectName
	} else {
		command = "npx wrangler pages deployment list --project-name " + projectName
	}

	_, err := runCommand(installDir, command)
	return err == nil
}

func extractURL(output string) (string, error) {
	url, err := findMatch(`https?://[^\s]+`, output)
	if err != nil {
		return "", fmt.Errorf("failed to get panel URL: %v", err)
	}

	return url, nil
}

func buildWranglerConfig(filePath string) error {

	config := map[string]any{
		"name":                projectName,
		"compatibility_date":  time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		"compatibility_flags": []string{"nodejs_compat"},
		"kv_namespaces": []map[string]string{
			{
				"binding": "kv",
				"id":      kvID,
			},
		},
		"vars": map[string]string{
			"UUID":     UUID,
			"TR_PASS":  TR_PASS,
			"PROXY_IP": PROXY_IP,
			"FALLBACK": FALLBACK,
			"SUB_PATH": SUB_PATH,
		},
	}

	if deployType == "1" {
		config["main"] = "./src/worker.js"
		config["workers_dev"] = true
	} else {
		config["pages_build_output_dir"] = "./src/"
	}

	if customDomain != "" {
		config["workers_dev"] = []map[string]any{
			{
				"custom_domain": true,
				"pattern":       customDomain,
			},
		}
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
		message += ": " + err.Error()
	}
	fmt.Printf("%s %s\n", errMark, message)
}

func successMessage(message string) {
	succMark := bold + green + "✓" + reset
	fmt.Printf("%s %s\n", succMark, message)
}

func launchPanel(output string) {
	prompt := fmt.Sprintf("Would you like to open %sBPB panel%s in browser? (y/n): ", blue, reset)
	if response := promptUser(prompt); strings.ToLower(response) == "n" {
		return
	}

	var panelURL string
	if customDomain == "" {
		if deployType == "1" {
			url, err := extractURL(output)
			if err != nil {
				failMessage("Error getting URL", err)
				return
			}
			panelURL = url + "/panel"
		} else {
			panelURL = "https://" + projectName + ".pages.dev/panel"
		}
	} else {
		panelURL = "https://" + customDomain + "/panel"
	}

	if err := openURL(panelURL); err != nil {
		failMessage("Error opening URL in browser", err)
		return
	}
}
