package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/kv"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/pages"
)

type projectDeploymentNewParams struct {
	AccountID string                `form:"account_id,required"`
	Branch    string                `form:"branch"`
	Manifest  string                `form:"manifest"`
	WorkerJS  *multipart.FileHeader `form:"_worker.js"`
	jsPath    string
}

func (pdp projectDeploymentNewParams) MarshalMultipart() ([]byte, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	manifestHeaders := textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="manifest"`},
	}

	manifestPart, err := writer.CreatePart(manifestHeaders)
	if err != nil {
		return nil, "", fmt.Errorf("error creating manifest part: %w", err)
	}

	_, err = manifestPart.Write([]byte("{}"))
	if err != nil {
		return nil, "", fmt.Errorf("error writing manifest content: %w", err)
	}

	branchHeaders := textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="branch"`},
	}

	branchPart, err := writer.CreatePart(branchHeaders)
	if err != nil {
		return nil, "", fmt.Errorf("error creating branch part: %w", err)
	}

	_, err = branchPart.Write([]byte("main"))
	if err != nil {
		return nil, "", fmt.Errorf("error writing branch content: %w", err)
	}

	fileHeaders := textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="_worker.js"; filename="_worker.js"`},
		"Content-Type":        []string{"application/javascript"},
	}

	filePart, err := writer.CreatePart(fileHeaders)
	if err != nil {
		return nil, "", fmt.Errorf("error creating file part: %w", err)
	}

	file, err := os.Open(pdp.jsPath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(filePart, file)
	if err != nil {
		return nil, "", fmt.Errorf("error copying file content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, "", fmt.Errorf("error closing multipart writer: %w", err)
	}

	return body.Bytes(), writer.FormDataContentType(), nil
}

func createPagesProject(
	ctx context.Context,
	name string,
	uid string,
	pass string,
	proxy string,
	fallback string,
	sub string,
	kv *kv.Namespace,
) (
	*pages.Project,
	error,
) {
	project, err := cfClient.Pages.Projects.New(
		ctx,
		pages.ProjectNewParams{
			AccountID: cf.F(cfAccount.ID),
			Project: pages.ProjectParam{
				Name:             cf.F(name),
				ProductionBranch: cf.F("main"),
				DeploymentConfigs: cf.F(pages.ProjectDeploymentConfigsParam{
					Production: cf.F(pages.ProjectDeploymentConfigsProductionParam{
						Browsers:           cf.F(map[string]pages.ProjectDeploymentConfigsProductionBrowserParam{}),
						CompatibilityDate:  cf.F(time.Now().AddDate(0, 0, -1).Format("2006-01-02")),
						CompatibilityFlags: cf.F([]string{"nodejs_compat"}),
						KVNamespaces: cf.F(map[string]pages.ProjectDeploymentConfigsProductionKVNamespaceParam{
							"kv": {
								NamespaceID: cf.F(kv.ID),
							},
						}),
						EnvVars: cf.F(map[string]pages.ProjectDeploymentConfigsProductionEnvVarsUnionParam{
							"UUID": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
								Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
								Value: cf.F(uid),
							},
							"TR_PASS": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
								Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
								Value: cf.F(pass),
							},
							"PROXY_IP": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
								Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
								Value: cf.F(proxy),
							},
							"FALLBACK": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
								Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
								Value: cf.F(fallback),
							},
							"SUB_PATH": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
								Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
								Value: cf.F(sub),
							},
						}),
					}),
				}),
			},
		})

	if err != nil {
		return nil, fmt.Errorf("error creating pages project: %w", err)
	}

	return project, nil
}

func createPagesDeployment(ctx context.Context, project *pages.Project) (*pages.Deployment, error) {
	param := projectDeploymentNewParams{
		AccountID: cfAccount.ID,
		Branch:    "main",
		Manifest:  "{}",
		WorkerJS:  &multipart.FileHeader{Filename: "worker.js"},
		jsPath:    workerPath,
	}
	data, ct, err := param.MarshalMultipart()
	if err != nil {
		return nil, fmt.Errorf("error marshalling pages multipart data: %w", err)
	}
	r := bytes.NewBuffer(data)

	deployment, err := cfClient.Pages.Projects.Deployments.New(
		ctx,
		project.Name,
		pages.ProjectDeploymentNewParams{AccountID: cf.F(cfAccount.ID)},
		option.WithRequestBody(ct, r),
	)

	if err != nil {
		return nil, fmt.Errorf("error creating pages deployment: %w", err)
	}

	return deployment, nil
}

func addPagesProjectCustomDomain(ctx context.Context, projectName string, customDomain string) (string, error) {
	// extractor, err := tldextract.New(cachePath, false)
	// if err != nil {
	// 	return "", fmt.Errorf("error extracting TLD: %w", err)
	// }

	// result := extractor.Extract(customDomain)
	// domain := fmt.Sprintf("%s.%s", result.Root, result.Tld)

	// zones, err := cfClient.Zones.List(ctx, zones.ZoneListParams{
	// 	Account: cf.F(zones.ZoneListParamsAccount{
	// 		ID: cf.F(cfAccount.ID),
	// 	}),
	// 	Match: cf.F(zones.ZoneListParamsMatch("contains")),
	// 	Name:  cf.F(domain),
	// })

	// if err != nil {
	// 	return "", err
	// }

	// if len(zones.Result) == 0 {
	// 	message := fmt.Sprintf("Could not find this domain in your account: %s", domain)
	// 	return "", fmt.Errorf(message, nil)
	// }

	// zone := zones.Result[0]
	// pagesHost := fmt.Sprintf("%s.pages.dev", projectName)

	// _, er := cfClient.DNS.Records.New(ctx, dns.RecordNewParams{
	// 	ZoneID: cf.F(zone.ID),
	// 	Record: dns.CNAMERecordParam{
	// 		Content: cf.F(pagesHost),
	// 		Name:    cf.F(customDomain),
	// 		Proxied: cf.F(true),
	// 		Type:    cf.F(dns.CNAMERecordType("CNAME")),
	// 	},
	// }, cfClient.Options...)

	// if er != nil {
	// 	return "", er
	// }

	res, err := cfClient.Pages.Projects.Domains.New(ctx, projectName, pages.ProjectDomainNewParams{
		AccountID: cf.F(cfAccount.ID),
		Name:      cf.F(customDomain),
	})

	if err != nil {
		return "", fmt.Errorf("error adding custom domain to pages: %w", err)
	}

	return res.Name, nil
}

func isPagesProjectAvailable(ctx context.Context, projectName string) bool {
	_, err := cfClient.Pages.Projects.Get(ctx, projectName, pages.ProjectGetParams{AccountID: cf.F(cfAccount.ID)})
	return err != nil
}

func listPages(ctx context.Context) ([]string, error) {
	projects, err := cfClient.Pages.Projects.List(ctx, pages.ProjectListParams{
		AccountID: cf.F(cfAccount.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing pages projects: %w", err)
	}

	if len(projects.Result) == 0 {
		return nil, fmt.Errorf("no pages projects found")
	}

	var projectNames []string
	for _, project := range projects.Result {
		rawName := project.JSON.ExtraFields["name"].Raw()
		var name string
		if err := json.Unmarshal([]byte(rawName), &name); err != nil {
			return nil, fmt.Errorf("error unmarshalling project name: %w", err)
		}

		projectNames = append(projectNames, name)
	}

	return projectNames, nil
}

func deletePagesProject(ctx context.Context, projectName string) error {
	domains, err := cfClient.Pages.Projects.Domains.List(
		ctx,
		projectName,
		pages.ProjectDomainListParams{AccountID: cf.F(cfAccount.ID)},
	)

	if err != nil {
		return fmt.Errorf("error listing project domains: %w", err)
	}

	if len(domains.Result) > 0 {
		fmt.Printf("\n%s Detaching custom domains...\n", title)
		for _, domain := range domains.Result {
			_, err := cfClient.Pages.Projects.Domains.Delete(
				ctx,
				projectName,
				domain.Name,
				pages.ProjectDomainDeleteParams{AccountID: cf.F(cfAccount.ID)},
			)
			if err != nil {
				return fmt.Errorf("error detaching custom domain: %w", err)
			}

			message := fmt.Sprintf("Custom domain %s detached successfully!", domain.Name)
			successMessage(message)
		}
	}

	_, er := cfClient.Pages.Projects.Delete(ctx, projectName, pages.ProjectDeleteParams{
		AccountID: cf.F(cfAccount.ID),
	})

	if er != nil {
		return fmt.Errorf("error deleting pages project: %w", er)
	}

	return nil
}

func updatePagesProject(ctx context.Context, projectName string) error {
	project, err := cfClient.Pages.Projects.Get(ctx, projectName, pages.ProjectGetParams{
		AccountID: cf.F(cfAccount.ID),
	})
	if err != nil {
		return fmt.Errorf("could not get project: %w", err)
	}

	param := projectDeploymentNewParams{
		AccountID: cfAccount.ID,
		Branch:    "main",
		Manifest:  "{}",
		WorkerJS:  &multipart.FileHeader{Filename: "worker.js"},
		jsPath:    workerPath,
	}
	data, ct, err := param.MarshalMultipart()
	if err != nil {
		return fmt.Errorf("error marshalling pages multipart data: %w", err)
	}
	r := bytes.NewBuffer(data)

	_, er := cfClient.Pages.Projects.Deployments.New(
		ctx,
		project.Name,
		pages.ProjectDeploymentNewParams{AccountID: cf.F(cfAccount.ID)},
		option.WithRequestBody(ct, r),
	)

	if er != nil {
		return fmt.Errorf("error updating pages project: %w", er)
	}

	return nil
}

func deployPagesProject(
	ctx context.Context,
	name string,
	uid string,
	pass string,
	proxy string,
	fallback string,
	sub string,
	kvNamespace *kv.Namespace,
	customDomain string,
) (
	panelURL string,
	er error,
) {
	var project *pages.Project
	var err error

	for {
		fmt.Printf("\n%s 正在创建 Pages 项目...\n", title)

		project, err = createPagesProject(ctx, name, uid, pass, proxy, fallback, sub, kvNamespace)
		if err != nil {
			failMessage("创建项目失败。")
			log.Printf("%v\n\n", err)
			if response := promptUser("是否重试？(y/n): "); strings.ToLower(response) == "n" {
				return "", nil
			}
			continue
		}

		successMessage("Page 创建成功！")
		break
	}

	for {
		fmt.Printf("\n%s 正在部署 Pages 项目...\n", title)

		_, err = createPagesDeployment(ctx, project)
		if err != nil {
			failMessage("部署项目失败。")
			log.Printf("%v\n\n", err)
			if response := promptUser("是否重试？(y/n): "); strings.ToLower(response) == "n" {
				return "", nil
			}
			continue
		}

		successMessage("Page 部署成功！")
		break
	}

	if customDomain != "" {
		for {
			recordName, err := addPagesProjectCustomDomain(ctx, name, customDomain)
			if err != nil {
				failMessage("添加自定义域名失败。")
				log.Printf("%v\n\n", err)
				if response := promptUser("是否重试？(y/n): "); strings.ToLower(response) == "n" {
					return "", nil
				}
				continue
			}

			successMessage("自定义域名添加成功！")
			fmt.Printf("%s %s: 你需要为 Name: %s 和 Target: %s 创建 CNAME 记录，否则自定义域名将无法生效。\n", info, warning, fmtStr(recordName, GREEN, true), fmtStr(name+".pages.dev", GREEN, true))
			return "https://" + customDomain + "/panel", nil
		}
	}

	successMessage("访问面板大约需要 5 分钟，请耐心等待...")
	return "https://" + project.Subdomain + "/panel", nil
}
