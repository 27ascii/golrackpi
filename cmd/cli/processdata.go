// Copyright 2022 Ralf Geschke. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/geschke/golrackpi"
	"github.com/spf13/cobra"

	"strings"
)

func init() {

	processdataGetCmd.Flags().BoolVarP(&csvOutput, "csv", "c", false, "Set output to CSV format")
	processdataGetCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "Set CSV delimiter (default \",\")")
	processdataMultCmd.Flags().BoolVarP(&csvOutput, "csv", "c", false, "Set output to CSV format")
	processdataMultCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "Set CSV delimiter (default \",\")")

	rootCmd.AddCommand(processdataCmd)
	processdataCmd.AddCommand(processdataListCmd)
	processdataCmd.AddCommand(processdataMultCmd)
	processdataCmd.AddCommand(processdataGetCmd)

}

var processdataCmd = &cobra.Command{
	Use: "processdata",

	Short: "List processdata values",
	//Long:  `Manage dynpower domain entries in database.`,
	Run: func(cmd *cobra.Command,
		args []string) {
		handleProcessdata()
	},
}

var processdataListCmd = &cobra.Command{
	Use: "list",

	Short: "List all available modules and processdata identifiers",
	//Long:  `List all domains in the dynpower database. If a DSN is submitted by the flag --dsn, this DSN will be used. If no DSN is provided, dynpower-cli tries to use the environment variables DBHOST, DBUSER, DBNAME and DBPASSWORD.`,

	Run: func(cmd *cobra.Command,
		args []string) {
		listProcessdata()
	},
}

var processdataMultCmd = &cobra.Command{
	Use: "mult [moduleid] [processdataid(s)] or get [moduleid|processdataid(s)] [moduleid|processdataid(s)] ... ",

	Short: "Get one or more modules with their processdata values",
	//Long:  `List all domains in the dynpower database. If a DSN is submitted by the flag --dsn, this DSN will be used. If no DSN is provided, dynpower-cli tries to use the environment variables DBHOST, DBUSER, DBNAME and DBPASSWORD.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command,
		args []string) {
		getMultProcessdata(args)
	},
}

var processdataGetCmd = &cobra.Command{
	Use: "get [moduleid] [processdataid(s)]",

	Short: "Get module with one or more of its processdata values",
	//Long:  `List all domains in the dynpower database. If a DSN is submitted by the flag --dsn, this DSN will be used. If no DSN is provided, dynpower-cli tries to use the environment variables DBHOST, DBUSER, DBNAME and DBPASSWORD.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command,
		args []string) {
		getProcessdata(args)
	},
}

func listProcessdata() {
	lib := golrackpi.NewWithParameter(golrackpi.AuthClient{
		Scheme:   authData.Scheme,
		Server:   authData.Server,
		Password: authData.Password,
	})

	_, err := lib.Login()
	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	defer lib.Logout()

	processData, err := lib.ProcessData()
	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	for _, pdItem := range processData {
		fmt.Println("ModuleId:", pdItem.ModuleId)
		if len(pdItem.ProcessDataIds) > 0 {
			fmt.Println("ProcessDataIds:")
			for _, pdId := range pdItem.ProcessDataIds {
				fmt.Println("\t", pdId)
			}
		} else {
			fmt.Println("No ProcessDataId found.")
		}

	}

}

func getMultProcessdata(args []string) {

	// check format of submitted arguments
	var requestProcessData []golrackpi.ProcessData

	if strings.Contains(args[0], "|") { // search "|"" separator to request one or more modules with their processdataids
		for _, argModuleProcessdata := range args {
			moduleProcessdata := strings.Split(argModuleProcessdata, "|")
			if len(moduleProcessdata) != 2 {
				fmt.Println("Wrong format of moduleid and processdataid values.")
				return
			}
			argModuleId := moduleProcessdata[0]
			processdataIds := strings.Split(moduleProcessdata[1], ",")
			v := golrackpi.ProcessData{ModuleId: argModuleId, ProcessDataIds: processdataIds}
			requestProcessData = append(requestProcessData, v)

		}

	} else if len(args) == 2 { // else moduleid and processdataids must submitted separately
		moduleIds := strings.Split(args[0], ",")
		processdataIds := strings.Split(args[1], ",")
		fmt.Println("moduleids:", moduleIds)
		fmt.Println("processdataids:", processdataIds)

		if len(moduleIds) > 1 {
			fmt.Println("Please enter only one moduleid.")
			return
		}
		v := golrackpi.ProcessData{ModuleId: moduleIds[0], ProcessDataIds: processdataIds}
		requestProcessData = append(requestProcessData, v)

	} else {
		fmt.Println("Please submit module and processdata in an appropriate format.")
		return
	}

	lib := golrackpi.NewWithParameter(golrackpi.AuthClient{
		Scheme:   authData.Scheme,
		Server:   authData.Server,
		Password: authData.Password,
	})

	_, err := lib.Login()
	if err != nil {
		fmt.Println("An error occurred:", err)

		return
	}
	defer lib.Logout()

	processDataValues, err := lib.ProcessDataValues(requestProcessData)
	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	if csvOutput {
		fmt.Printf("Module%sProcessdata Id%sProcessdata Unit%sProcessdata Value\n", delimiter, delimiter, delimiter)
		for _, pdv := range processDataValues {
			for _, pd := range pdv.ProcessData {

				fmt.Printf("%s%s%s%s%s%s%v\n", pdv.ModuleId, delimiter, pd.Id, delimiter, pd.Unit, delimiter, pd.Value)

			}
		}

	} else {

		for _, pdv := range processDataValues {
			fmt.Println("Module:", pdv.ModuleId)
			for _, pd := range pdv.ProcessData {
				fmt.Println(pd.Id, "\t", pd.Unit, "\t", pd.Value)
			}
			fmt.Println()
		}

	}
}

func getProcessdata(args []string) {

	// submitted values: moduleid pdid pdid2 pdid3...

	moduleId := args[0]
	processDataIds := args[1:]

	lib := golrackpi.NewWithParameter(golrackpi.AuthClient{
		Scheme:   authData.Scheme,
		Server:   authData.Server,
		Password: authData.Password,
	})

	_, err := lib.Login()
	if err != nil {
		fmt.Println("An error occurred:", err)

		return
	}
	defer lib.Logout()

	processDataValues, err := lib.ProcessDataModuleValues(moduleId, processDataIds...)
	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	if csvOutput {
		fmt.Printf("Module%sProcessdata Id%sProcessdata Unit%sProcessdata Value\n", delimiter, delimiter, delimiter)
		for _, pdv := range processDataValues {
			for _, pd := range pdv.ProcessData {

				fmt.Printf("%s%s%s%s%s%s%v\n", pdv.ModuleId, delimiter, pd.Id, delimiter, pd.Unit, delimiter, pd.Value)

			}
		}

	} else {

		for _, pdv := range processDataValues {
			fmt.Println("Module:", pdv.ModuleId)
			for _, pd := range pdv.ProcessData {
				fmt.Println(pd.Id, "\t", pd.Unit, "\t", pd.Value)
			}
			fmt.Println()
		}

	}
}

/*
* Handle processdata-related commands
 */
func handleProcessdata() {
	fmt.Println("\nUnknown or missing command.\nRun golrackpi processdata --help to show available commands.")
}
