package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
)

type Download struct {
	repo    string
	chart   string
	version string
}

type HelmRepo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// CheckHelmRepoExists vérifie si un repository Helm existe.
func CheckHelmRepoExists(repoName string) bool {
	cmd := exec.Command("helm", "repo", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Erreur lors de la vérification du repository Helm :", err)
		return false
	}

	// Analyser la sortie JSON
	var repos []HelmRepo
	err = json.Unmarshal(output, &repos)
	if err != nil {
		fmt.Println("Erreur lors de l'analyse de la sortie JSON :", err)
		return false
	}

	// Vérifier si le repository existe dans la liste
	for _, repo := range repos {
		if repo.Name == repoName {
			return true
		}
	}
	return false
}

func getHelmChartVersions(repo, chart string) ([]string, error) {
	cmd := exec.Command("helm", "search", "repo", fmt.Sprintf("%s/%s", repo, chart), "--versions", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute helm search: %v", err)
	}

	// Analyser la sortie JSON
	var chartVersions []struct {
		Version string `json:"version"`
	}
	err = json.Unmarshal(output, &chartVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse helm search output: %v", err)
	}

	// Extraire les versions disponibles
	var versions []string
	for _, cv := range chartVersions {
		versions = append(versions, cv.Version)
	}

	return versions, nil
}

func fetchHelmChart(chartName, version, destination string) error {
	// Exécute la commande helm fetch
	if destination == "" {
		destination = "."
	}
	cmd := exec.Command("helm", "fetch", chartName, "--version", version, "--untar", "--untardir", destination)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute helm fetch: %v", err)
	}

	fmt.Printf("Helm chart fetched successfully to %s\n", destination)
	return nil
}

func createOptionsFromStrings(strings []string) []huh.Option[string] {
	var options []huh.Option[string]
	for _, str := range strings {
		options = append(options, huh.NewOption(string(str), string(str)))
	}
	return options
}

func getForm[T comparable](selects ...*huh.Select[T]) *huh.Form {
	var fields []huh.Field
	for _, sel := range selects {
		fields = append(fields, sel)
	}
	group := huh.NewGroup(fields...)
	return huh.NewForm(group)
}

func main() {
	var dl Download

	// firstForm := huh.NewForm(
	// 	huh.NewGroup(
	// 		// Ask the user for a base burger and toppings.
	// 		huh.NewSelect[string]().
	// 			Title("Select repository").
	// 			Options(
	// 				huh.NewOption("Bitnami", "bitnami"),
	// 				huh.NewOption("QUAY IO", "quay.io"),
	// 			).
	// 			Value(&dl.repo), // store the chosen option in the "repo" variable

	// 		// Let the user select multiple toppings.
	// 		huh.NewSelect[string]().
	// 			Title("select Helm chart from repository").
	// 			Options(
	// 				huh.NewOption("NGINX", "nginx"),
	// 				huh.NewOption("POSTGRESQL", "postgres"),
	// 			).
	// 			Value(&dl.chart),
	// 	),
	// )

	firstForm := getForm(
		huh.NewSelect[string]().Title("Select repository").Options(huh.NewOption("Bitnami", "bitnami"), huh.NewOption("QUAY IO", "quay.io")).Value(&dl.repo),
		huh.NewSelect[string]().Title("select Helm chart from repository").Options(huh.NewOption("NGINX", "nginx"), huh.NewOption("POSTGRESQL", "postgres")).Value(&dl.chart),
	)

	err := firstForm.Run()
	if err != nil {
		log.Fatal(err)
	}

	exists := CheckHelmRepoExists(dl.repo)

	if exists {
		fmt.Printf("Le repository Helm '%s' existe.\n", dl.repo)
	} else {
		fmt.Printf("Erreur le repository Helm '%s' n'existe pas.\n", dl.repo)
		os.Exit(1)
	}

	versions, err := getHelmChartVersions(dl.repo, dl.chart)
	if err != nil {
		fmt.Printf("Erreur lors de la récupération des versions du Helm chart : %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Versions disponibles :")
	for _, v := range versions {
		fmt.Println(v)
	}

	// versionForm := huh.NewForm(
	// 	huh.NewGroup(
	// 		huh.NewSelect[string]().
	// 			Title("which version do you want ?").
	// 			Options(createOptionsFromStrings(versions)...).
	// 			Value(&dl.version),
	// 	),
	// )

	versionForm := getForm(
		huh.NewSelect[string]().Title("which version do you want ?").Options(createOptionsFromStrings(versions)...).Value(&dl.version),
	)

	err = versionForm.Run()
	if err != nil {
		log.Fatal(err)
	}

	chartFullName := fmt.Sprintf("%s/%s", dl.repo, dl.chart)

	err = fetchHelmChart(chartFullName, dl.version, "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

}
