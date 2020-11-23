package cobra

import (
	"log"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewInvalidateCommand(cmd *Command) {

	cobraCmd := &cobra.Command{
		Use:   "invalidate",
		Short: "Invalidate CloudFront",

		Run: func(cobraCmd *cobra.Command, args []string) {
			invalidate(cmd)
		},
		PreRun: func(cobraCmd *cobra.Command, args []string) {
			cmd.CheckEnv()
			cmd.GetAWSSession()
		},
	}

	cobraCmd.Flags().StringVarP(&cmd.GTenv, "env", "e", "", "Environment to use")

	cobraCmd.Flags().StringSliceVarP(&cmd.SelectedServices, "service", "s", []string{}, "Service(s) to show. Separated by comma")

	cmd.AddCommand(cobraCmd)

}

func invalidate(cmd *Command) {
	cmd.AWSSession.GetCloudFronts(&cmd.CloudFronts, cmd.GTenv)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Cloudfront ID", "Pattern", "Associated Service", "invalidate ID", "Status"})

	if len(cmd.SelectedServices) <= 0 {
		// Process all (unfiltered) CF invalidations description (even without service associated)
		for _, cf := range cmd.CloudFronts.CloudFronts {
			if !strings.EqualFold("", cf.CloudfrontID) && !cf.IgnoreDeploy {
				// fmt.Printf("CF ID: %s Pattern: %s associated Service: %s Ignore: %t\n", cf.CloudfrontID, cf.CloudFrontPattern, cf.AssociatedService, cf.IgnoreDeploy)
				resp, err := cmd.AWSSession.CreateInvalidationRequest(cf.CloudfrontID, cf.CloudFrontPattern)
				if err != nil {
					log.Fatalf("Invalidate Failed. %v", err)
				}
				t.AppendRow([]interface{}{
					cf.CloudfrontID,
					cf.CloudFrontPattern,
					cf.AssociatedService,
					*resp.Invalidation.Id,
					*resp.Invalidation.Status,
				})
			}
		}
	} else {
		//Process Filtered CF invalidations (With service associated)
		for _, cf := range cmd.CloudFronts.CloudFronts {
			for _, s := range cmd.SelectedServices {
				if strings.EqualFold(s, cf.AssociatedService) && !strings.EqualFold("", cf.CloudfrontID) && !cf.IgnoreDeploy {
					resp, err := cmd.AWSSession.CreateInvalidationRequest(cf.CloudfrontID, cf.CloudFrontPattern)
					if err != nil {
						log.Fatalf("Invalidate Failed. %v", err)
					}
					t.AppendRow([]interface{}{
						cf.CloudfrontID,
						cf.CloudFrontPattern,
						cf.AssociatedService,
						*resp.Invalidation.Id,
						*resp.Invalidation.Status,
					})
				}
			}
		}
	}

	switch cmd.TableStyle {
	case "light":
		t.SetStyle(table.StyleLight)
	case "color":
		t.SetStyle(table.StyleColoredDark)
	}
	if t.Length() > cmd.ShowTableIndexAbove {
		t.SetAutoIndex(true)
	}

	t.Render()

}
