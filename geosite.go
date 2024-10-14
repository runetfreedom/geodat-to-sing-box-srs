package main

import (
	"github.com/sagernet/sing-box/common/geosite"
	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/spf13/cobra"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	rootCmd.AddCommand(geositeCmd)
	geositeCmd.PersistentFlags().StringP("input", "i", "geosite.dat", "Path or URL to geosite.dat")
	geositeCmd.PersistentFlags().StringP("output_dir", "o", "rules", "Output directory")
	geositeCmd.PersistentFlags().StringP("prefix", "p", "geosite-", "Output file name prefix")
}

var geositeCmd = &cobra.Command{
	Use:   "geosite",
	Short: "Convert geosite.dat to srs",
	Run: func(cmd *cobra.Command, args []string) {
		inputFile, _ := cmd.Flags().GetString("input")
		outDir, _ := cmd.Flags().GetString("output_dir")
		prefix, _ := cmd.Flags().GetString("prefix")
		log.Println("Use source:", inputFile)

		geositeBytes, err := readFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		err = os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		var geositeList routercommon.GeoSiteList
		if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
			log.Fatal(err)
		}

		domainMap := make(map[string][]geosite.Item)
		for _, vGeositeEntry := range geositeList.Entry {
			code := strings.ToLower(vGeositeEntry.CountryCode)
			domains := make([]geosite.Item, 0, len(vGeositeEntry.Domain)*2)
			attributes := make(map[string][]*routercommon.Domain)
			for _, domain := range vGeositeEntry.Domain {
				if len(domain.Attribute) > 0 {
					for _, attribute := range domain.Attribute {
						attributes[attribute.Key] = append(attributes[attribute.Key], domain)
					}
				}
				switch domain.Type {
				case routercommon.Domain_Plain:
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomainKeyword,
						Value: domain.Value,
					})
				case routercommon.Domain_Regex:
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomainRegex,
						Value: domain.Value,
					})
				case routercommon.Domain_RootDomain:
					if strings.Contains(domain.Value, ".") {
						domains = append(domains, geosite.Item{
							Type:  geosite.RuleTypeDomain,
							Value: domain.Value,
						})
					}
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomainSuffix,
						Value: "." + domain.Value,
					})
				case routercommon.Domain_Full:
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomain,
						Value: domain.Value,
					})
				}
			}
			domainMap[code] = common.Uniq(domains)
			for attribute, attributeEntries := range attributes {
				attributeDomains := make([]geosite.Item, 0, len(attributeEntries)*2)
				for _, domain := range attributeEntries {
					switch domain.Type {
					case routercommon.Domain_Plain:
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomainKeyword,
							Value: domain.Value,
						})
					case routercommon.Domain_Regex:
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomainRegex,
							Value: domain.Value,
						})
					case routercommon.Domain_RootDomain:
						if strings.Contains(domain.Value, ".") {
							attributeDomains = append(attributeDomains, geosite.Item{
								Type:  geosite.RuleTypeDomain,
								Value: domain.Value,
							})
						}
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomainSuffix,
							Value: "." + domain.Value,
						})
					case routercommon.Domain_Full:
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomain,
							Value: domain.Value,
						})
					}
				}
				domainMap[code+"@"+attribute] = common.Uniq(attributeDomains)
			}
		}

		for countryCode, domains := range domainMap {
			var headlessRule option.DefaultHeadlessRule
			defaultRule := geosite.Compile(domains)
			headlessRule.Domain = defaultRule.Domain
			headlessRule.DomainSuffix = defaultRule.DomainSuffix
			headlessRule.DomainKeyword = defaultRule.DomainKeyword
			headlessRule.DomainRegex = defaultRule.DomainRegex
			var plainRuleSet option.PlainRuleSet
			plainRuleSet.Rules = []option.HeadlessRule{
				{
					Type:           C.RuleTypeDefault,
					DefaultOptions: headlessRule,
				},
			}

			srsPath, _ := filepath.Abs(filepath.Join(outDir, prefix+countryCode+".srs"))
			outFile, err := os.Create(srsPath)
			if err != nil {
				log.Fatal(err)
			}
			err = srs.Write(outFile, plainRuleSet)
			outFile.Close()
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}
