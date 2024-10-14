package main

import (
	"fmt"
	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/spf13/cobra"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	rootCmd.AddCommand(geoipCmd)
	geoipCmd.PersistentFlags().StringP("input", "i", "geoip.dat", "Path or URL to geoip.dat")
	geoipCmd.PersistentFlags().StringP("output_dir", "o", "rules", "Output directory")
	geoipCmd.PersistentFlags().StringP("prefix", "p", "geoip-", "Output file name prefix")
}

var geoipCmd = &cobra.Command{
	Use:   "geoip",
	Short: "Convert geoip.dat to srs",
	Run: func(cmd *cobra.Command, args []string) {
		inputFile, _ := cmd.Flags().GetString("input")
		outDir, _ := cmd.Flags().GetString("output_dir")
		prefix, _ := cmd.Flags().GetString("prefix")
		log.Println("Use source:", inputFile)

		geoipBytes, err := readFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		err = os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		var geoipList routercommon.GeoIPList
		if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
			log.Fatal(err)
		}

		for _, geoip := range geoipList.Entry {
			countryCode := strings.ToLower(geoip.GetCountryCode())
			if countryCode == "" {
				continue
			}

			var headlessRule option.DefaultHeadlessRule
			headlessRule.IPCIDR = make([]string, 0, len(geoip.Cidr))

			for _, v2rayCIDR := range geoip.Cidr {
				headlessRule.IPCIDR = append(headlessRule.IPCIDR, net.IP(v2rayCIDR.GetIp()).String()+"/"+fmt.Sprint(v2rayCIDR.GetPrefix()))
			}

			headlessRule.IPCIDR = common.Uniq(headlessRule.IPCIDR)

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
