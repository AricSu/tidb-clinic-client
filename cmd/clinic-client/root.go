package main

import (
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type commandDeps struct {
	runCloudMetricsQueryRange func() error
	runOPMetricsQueryRange    func() error
	runOPSlowQueriesQuery     func() error
	runOPLogsSearch           func() error
	runOPConfigsGet           func() error
	runCloudClusterDetail     func() error
	runCloudEventsQuery       func() error
	runCloudEventsDetail      func() error
	runCloudTopSQLSummary     func() error
	runCloudTopSlowQueries    func() error
	runCloudSlowQueriesList   func() error
	runCloudSlowQueriesDetail func() error
}

func defaultCommandDeps() commandDeps {
	return commandDeps{
		runCloudMetricsQueryRange: func() error {
			return runMetricsQueryRange(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runOPMetricsQueryRange: func() error {
			return runOPMetricsQueryRange(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runOPSlowQueriesQuery: func() error {
			return runOPSlowQueriesQuery(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runOPLogsSearch: func() error {
			return runOPLogsSearch(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runOPConfigsGet: func() error {
			return runOPConfigsGet(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudClusterDetail: func() error {
			return runCloudClusterDetail(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudEventsQuery: func() error {
			return runCloudEventsQuery(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudEventsDetail: func() error {
			return runCloudEventsDetail(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudTopSQLSummary: func() error {
			return runCloudTopSQLSummary(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudTopSlowQueries: func() error {
			return runCloudTopSlowQueries(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudSlowQueriesList: func() error {
			return runCloudSlowQueriesList(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
		runCloudSlowQueriesDetail: func() error {
			return runCloudSlowQueriesDetail(os.LookupEnv, time.Now, log.Default(), os.Stdout)
		},
	}
}

func newRootCommand() *cobra.Command {
	return newRootCommandWithDeps(defaultCommandDeps())
}

func newRootCommandWithDeps(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "clinic-client",
		Short:         "TiDB Clinic CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newOPCommand(deps))
	cmd.AddCommand(newCloudCommand(deps))
	return cmd
}
