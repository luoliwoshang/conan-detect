package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type pkgInfo struct {
	pkgName      string
	firstVersion string //first version in conandata.yml
	firstUrls    []string
}

func (p *pkgInfo) String() string {
	return fmt.Sprintf("Package: %s\nVersion: %s\nURLs: %v\n", p.pkgName, p.firstVersion, p.firstUrls)
}

var rootCmd = &cobra.Command{
	Use:   "check-conan-info",
	Short: "Read package data information",
	Long:  `This tool is used to read version information and parse sources data from conandata.yml files.`,
}

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read information of the specified package",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		packageName := args[0]
		directory, _ := cmd.Flags().GetString("dir")
		readPackageInfo(packageName, directory)
	},
}

var listCmd = &cobra.Command{
	Use:   "list [start] [count]",
	Short: "List information for multiple packages",
	Long:  `List information for multiple packages, starting from the specified position and showing the specified number of packages.`,
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		directory, _ := cmd.Flags().GetString("dir")

		var start, count int
		var err error

		if all {
			start = 0
			count = -1
		} else {
			if len(args) != 2 {
				fmt.Println("Error: requires start and count arguments when --all is not specified")
				return
			}

			start, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid start position: %v\n", err)
				return
			}

			count, err = strconv.Atoi(args[1])
			if err != nil {
				fmt.Printf("Invalid count: %v\n", err)
				return
			}
		}

		countGithub, _ := cmd.Flags().GetBool("count-github")
		listPackageInfo(start, count, countGithub, directory)
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(listCmd)

	addDirFlag := func(cmd *cobra.Command) {
		cmd.Flags().String("dir", "temp-ver", "Directory containing package information")
	}

	addDirFlag(readCmd)
	addDirFlag(listCmd)

	listCmd.Flags().Bool("count-github", false, "Count how many packages have their first URL starting with https://github.com/")
	listCmd.Flags().Bool("all", false, "Process all packages in temp-ver directory")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readPackageInfo(packageName, directory string) {
	pkgInfo, err := readPackageInfoWithReturn(packageName, directory)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(pkgInfo)
}

func readPackageInfoWithReturn(packageName, directory string) (*pkgInfo, error) {
	pkgInfo := &pkgInfo{
		pkgName: packageName,
	}
	packageDir := filepath.Join(directory, packageName)

	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("package %s does not exist", packageName)
	}

	versions, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read version directories: %v", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for package %s", packageName)
	}

	var firstVersion os.DirEntry
	for _, v := range versions {
		if v.IsDir() {
			firstVersion = v
			break
		}
	}

	if firstVersion == nil {
		return nil, fmt.Errorf("no version directories found for package %s", packageName)
	}

	versionName := firstVersion.Name()

	// Read data.path file
	dataPathFile := filepath.Join(packageDir, versionName, "data.path")
	filePath, err := readFilePath(dataPathFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read data.path: %v %s", err, dataPathFile)
	}

	// Read content of the specified file
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %v %s", err, filePath)
	}

	// Parse YAML content
	var node yaml.Node
	err = yaml.Unmarshal([]byte(content), &node)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML content: %v %s", err, filePath)
	}

	var root *yaml.Node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		root = node.Content[0]
	}

	sources, err := getValueByKey(root, "sources")
	if err != nil {
		return nil, fmt.Errorf("failed to get sources: %v", err)
	}

	pkgInfo.firstVersion = sources.Content[0].Value
	versionMap := sources.Content[1]
	if versionMap.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("sources first version is not a mapping node: %v", versionMap)
	}

	urlNode, err := getValueByKey(versionMap, "url")
	if err != nil {
		return nil, fmt.Errorf("failed to get url: %v", err)
	}

	switch urlNode.Kind {
	case yaml.SequenceNode:
		for _, url := range urlNode.Content {
			pkgInfo.firstUrls = append(pkgInfo.firstUrls, url.Value)
		}
	case yaml.ScalarNode:
		pkgInfo.firstUrls = append(pkgInfo.firstUrls, urlNode.Value)
	default:
		return nil, fmt.Errorf("unsupported URL format: %v", urlNode.Kind)
	}
	return pkgInfo, nil
}

func readFilePath(pathFile string) (string, error) {
	file, err := os.Open(pathFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("data.path is empty")
}

func readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func getValueByKey(node *yaml.Node, key string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("node is not a mapping node: %v", node)
	}

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("key not found: %s", key)
}

func listPackageInfo(start, count int, countGithub bool, directory string) {
	packages, err := os.ReadDir(directory)
	if err != nil {
		fmt.Printf("Failed to read packages directory: %v\n", err)
		return
	}

	if len(packages) == 0 {
		fmt.Println("No packages found")
		return
	}

	if start < 0 || start >= len(packages) {
		fmt.Printf("Invalid start position: %d (total packages: %d)\n", start, len(packages))
		return
	}

	end := len(packages)
	if count >= 0 && start+count < end {
		end = start + count
	}

	fmt.Printf("Listing packages %d to %d (total: %d) in directory: %s\n\n", start+1, end, len(packages), directory)

	githubCount := 0
	errorCount := 0

	for i := start; i < end; i++ {
		pkg := packages[i]
		if pkg.IsDir() {
			fmt.Printf("=== Package %d: %s ===\n", i+1, pkg.Name())
			pkgInfo, err := readPackageInfoWithReturn(pkg.Name(), directory)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				errorCount++
				fmt.Println()
				continue
			}
			fmt.Println(pkgInfo)

			if len(pkgInfo.firstUrls) > 0 && strings.HasPrefix(pkgInfo.firstUrls[0], "https://github.com/") {
				githubCount++
			}
		}
	}

	if countGithub {
		fmt.Printf("\n=== Statistics ===\n")
		fmt.Printf("Packages with first URL starting with https://github.com/: %d/%d\n",
			githubCount, end-start)
		fmt.Printf("Packages with errors: %d/%d\n",
			errorCount, end-start)
	}
}
