package cobra

import (
	"log"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewListInvalidationCommand(cmd *Command) {

	cobraCmd := &cobra.Command{
		Use:   "list-invalidation",
		Short: "List Cloudfront Invalidations",

		Run: func(cobraCmd *cobra.Command, args []string) {
			listinvalidation(cmd)
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

func listinvalidation(cmd *Command) {
	cmd.AWSSession.GetCloudFronts(&cmd.CloudFronts, cmd.GTenv)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Cloudfront ID", "Pattern", "Associated Service", "invalidate ID", "Date", "Status"})

	if len(cmd.SelectedServices) <= 0 {
		// Process all (unfiltered) CF invalidations description (even without service associated)
		for _, cf := range cmd.CloudFronts.CloudFronts {
			if !strings.EqualFold("", cf.CloudfrontID) && !cf.IgnoreDeploy {
				// fmt.Printf("CF ID: %s Pattern: %s associated Service: %s Ignore: %t\n", cf.CloudfrontID, cf.CloudFrontPattern, cf.AssociatedService, cf.IgnoreDeploy)
				resp, err := cmd.AWSSession.ViewListInvalidations(cf.CloudfrontID)
				if err != nil {
					log.Fatalf("List Invalidation Failed. %v", err)
				}

				for _, item := range resp.InvalidationList.Items {

					invalidation, err := cmd.AWSSession.GetInvalidationRequest(cf.CloudfrontID, *item.Id)
					if err != nil {
						log.Fatal(err)
					}

					t.AppendRow([]interface{}{
						cf.CloudfrontID,
						*invalidation.Invalidation.InvalidationBatch.Paths.Items[0],
						//cf.CloudFrontPattern,
						cf.AssociatedService,
						*item.Id,
						item.CreateTime,
						*item.Status,
					})

				}
			}
		}
	} else {
		//Process Filtered CF invalidations (With service associated)
		for _, cf := range cmd.CloudFronts.CloudFronts {
			for _, s := range cmd.SelectedServices {
				if strings.EqualFold(s, cf.AssociatedService) && !strings.EqualFold("", cf.CloudfrontID) && !cf.IgnoreDeploy {
					resp, err := cmd.AWSSession.ViewListInvalidations(cf.CloudfrontID)
					if err != nil {
						log.Fatalf("List Invalidation Failed. %v", err)
					}
					for _, item := range resp.InvalidationList.Items {
						invalidation, err := cmd.AWSSession.GetInvalidationRequest(cf.CloudfrontID, *item.Id)
						if err != nil {
							log.Fatal(err)
						}
						t.AppendRow([]interface{}{
							cf.CloudfrontID,
							*invalidation.Invalidation.InvalidationBatch.Paths.Items[0],
							// cf.CloudFrontPattern,
							cf.AssociatedService,
							*item.Id,
							item.CreateTime,
							*item.Status,
						})

					}
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
