package cmd

import (
	"github.com/olekukonko/tablewriter"
	hubtypes "github.com/sentinel-official/hub/types"
	"github.com/spf13/cobra"

	"github.com/sentinel-official/cli-client/context"
	clitypes "github.com/sentinel-official/cli-client/types"
	"github.com/sentinel-official/cli-client/x/provider/types"
)

var (
	header = []string{
		"Name",
		"Address",
		"Identity",
		"Website",
	}
)

func QueryProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			qc, err := context.NewQueryContextFromCmd(cmd)
			if err != nil {
				return err
			}

			provAddr, err := hubtypes.ProvAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			result, err := qc.QueryProvider(provAddr)
			if err != nil {
				return err
			}

			var (
				item  = types.NewProviderFromRaw(result)
				table = tablewriter.NewWriter(cmd.OutOrStdout())
			)

			table.SetHeader(header)
			table.Append(
				[]string{
					item.Name,
					item.Address,
					item.Identity,
					item.Website,
				},
			)

			table.Render()
			return nil
		},
	}

	clitypes.AddQueryFlagsToCmd(cmd)

	return cmd
}

func QueryProviders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Query providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			qc, err := context.NewQueryContextFromCmd(cmd)
			if err != nil {
				return err
			}

			pagination, err := clitypes.GetPageRequestFromCmd(cmd)
			if err != nil {
				return err
			}

			result, err := qc.QueryProviders(pagination)
			if err != nil {
				return err
			}

			var (
				items = types.NewProvidersFromRaw(result)
				table = tablewriter.NewWriter(cmd.OutOrStdout())
			)

			table.SetHeader(header)
			for i := 0; i < len(items); i++ {
				table.Append(
					[]string{
						items[i].Name,
						items[i].Address,
						items[i].Identity,
						items[i].Website,
					},
				)
			}

			table.Render()
			return nil
		},
	}

	clitypes.AddQueryFlagsToCmd(cmd)
	clitypes.AddPaginationFlagsToCmd(cmd, "providers")

	return cmd
}
