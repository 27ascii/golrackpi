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

	settingsModuleCmd.Flags().BoolVarP(&outputCSV, "csv", "c", false, "Set output to CSV format")
	settingsModuleCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "Set CSV delimiter (default \",\")")
	settingsModuleSettingCmd.Flags().BoolVarP(&outputCSV, "csv", "c", false, "Set output to CSV format")
	settingsModuleSettingCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "Set CSV delimiter (default \",\")")
	settingsModuleSettingsCmd.Flags().BoolVarP(&outputCSV, "csv", "c", false, "Set output to CSV format")
	settingsModuleSettingsCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ",", "Set CSV delimiter (default \",\")")

	rootCmd.AddCommand(settingsCmd)
	settingsCmd.AddCommand(settingsListCmd)
	settingsCmd.AddCommand(settingsModuleCmd)
	settingsCmd.AddCommand(settingsModuleSettingCmd)
	settingsCmd.AddCommand(settingsModuleSettingsCmd)

}

var settingsCmd = &cobra.Command{
	Use: "settings",

	Short: "List settings content",
	//Long:  ``,
	Run: func(cmd *cobra.Command,
		args []string) {
		handleSettings()
	},
}

var settingsListCmd = &cobra.Command{
	Use: "list",

	Short: "List all modules with their list of settings identifiers.",
	//Long:  ``,

	Run: func(cmd *cobra.Command,
		args []string) {
		listSettings()
	},
}

var settingsModuleCmd = &cobra.Command{
	Use: "module <moduleid>",

	Short: "Get module settings values.",
	//Long:  ``,

	Run: func(cmd *cobra.Command,
		args []string) {
		getSettingsModule(args)
	},
}

var settingsModuleSettingCmd = &cobra.Command{
	Use: "setting <moduleid> <settingid>",

	Short: "Get module setting value.",
	//Long:  ``,

	Run: func(cmd *cobra.Command,
		args []string) {
		getSettingsModuleSetting(args)
	},
}

var settingsModuleSettingsCmd = &cobra.Command{
	Use: "settings <moduleid> <settingids>",

	Short: "Get module settings values. Use a comma-separated list of settingids.",
	//Long:  ``,

	Run: func(cmd *cobra.Command,
		args []string) {
		getSettingsModuleSettings(args)
	},
}

// listSettings prints a (huge) list of module ids with their corresponding setting ids
func listSettings() {
	lib := golrackpi.NewWithParameter(golrackpi.AuthClient{
		Scheme:   authData.Scheme,
		Server:   authData.Server,
		Password: authData.Password,
	})

	_, err := lib.Login()
	defer lib.Logout()

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	settings, err := lib.Settings()

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	for _, s := range settings {
		fmt.Println(s.ModuleId)
		for _, data := range s.Settings {
			fmt.Println("\t", data.Id)
		}
	}

}

// getSettingsModule takes a module id as argument and prints setting ids and their current values
func getSettingsModule(args []string) {

	if len(args) < 1 {
		fmt.Println("Please submit a moduleid.")
		return
	} else if len(args) > 1 {
		fmt.Println("Please submit only one moduleid.")
		return
	}

	moduleId := args[0]

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

	values, err := lib.SettingsModule(moduleId)

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	writeSettingValues(values)
}

// getSettingsModuleSetting takes a module id and a setting id as arguments and prints setting ids and their current value
func getSettingsModuleSetting(args []string) {

	if len(args) < 2 {
		fmt.Println("Please submit a moduleid and a settingid.")
		return
	} else if len(args) > 2 {
		fmt.Println("Please submit only one moduleid with its settingid.")
		return
	}

	moduleId := args[0]
	settingId := args[1]

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

	values, err := lib.SettingsModuleSetting(moduleId, settingId)

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	writeSettingValues(values)

}

// getSettingsModuleSettings takes a module id and one or more setting ids as arguments and prints setting ids and their current value
func getSettingsModuleSettings(args []string) {

	if len(args) < 2 {
		fmt.Println("Please submit a moduleid and s comma-separated list of settingids.")
		return
	} else if len(args) > 2 {
		fmt.Println("Please submit only one moduleid with its settingids as comma-separated list.")
		return
	}

	settingIds := strings.Split(args[1], ",")

	moduleId := args[0]

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

	values, err := lib.SettingsModuleSettings(moduleId, settingIds)

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	writeSettingValues(values)

}

// writeSettingValues is a helper function to print a slice of setting ids and their value
func writeSettingValues(values []golrackpi.SettingsValues) {

	if outputCSV {
		fmt.Printf("Id%sValue\n", delimiter)
		for _, v := range values {
			fmt.Printf("%s%s%s\n", v.Id, delimiter, v.Value)
		}
	} else {
		fmt.Println("Id\tValue")
		for _, v := range values {
			fmt.Printf("%s\t%s\n", v.Id, v.Value)
		}

	}

}

/*
* Handle settings-related commands
 */
func handleSettings() {
	fmt.Println("\nUnknown or missing command.\nRun golrackpi settings --help to show available commands.")
}
